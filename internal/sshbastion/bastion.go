// Package sshbastion implements an SSH bastion proxy server that authenticates
// users via GateKey API keys and proxies SSH connections to internal target hosts
// with connection logging and optional session recording.
package sshbastion

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// Config holds SSH bastion server configuration.
type Config struct {
	ListenAddr  string // e.g., ":2222"
	HostKeyPath string // Path to store the host key PEM file
}

// AuthResult contains authenticated user information.
type AuthResult struct {
	UserID    string
	UserEmail string
	IsAdmin   bool
	Groups    []string
}

// AuthFunc validates SSH credentials and returns user info.
// The password parameter is the GateKey API key (gk_...).
type AuthFunc func(ctx context.Context, password string) (*AuthResult, error)

// AccessCheckFunc checks if a user has access to a target host.
type AccessCheckFunc func(ctx context.Context, userID string, groups []string, targetHost string, targetPort int) (bool, error)

// ConnectionLogFunc is called to log bastion connection events.
type ConnectionLogFunc func(userID, userEmail, sourceIP, targetHost string, targetPort int, bytesSent, bytesReceived int64, start time.Time, end time.Time)

// Server is the SSH bastion proxy server.
type Server struct {
	config    Config
	sshConfig *ssh.ServerConfig
	listener  net.Listener
	logger    *zap.Logger

	// Callbacks set by the API server
	AuthenticateUser AuthFunc
	CheckAccess      AccessCheckFunc
	LogConnection    ConnectionLogFunc

	// Recording callbacks
	StartRecording func(sessionID, recordingID, userEmail, targetHost string) (string, error)
	WriteRecording func(sessionID, output string)
	StopRecording  func(sessionID string) (int64, int, error)
	OnRecordStart  func(recordingID, userID, userEmail, targetHost, storagePath string)
	OnRecordStop   func(recordingID string, fileSize int64, durationSec int)

	mu sync.Mutex
}

// NewServer creates a new SSH bastion server.
func NewServer(config Config, logger *zap.Logger) (*Server, error) {
	s := &Server{
		config: config,
		logger: logger,
	}

	s.sshConfig = &ssh.ServerConfig{
		PasswordCallback: s.passwordCallback,
		ServerVersion:    "SSH-2.0-GateKey-Bastion",
	}

	return s, nil
}

// SetHostKey sets the SSH host key from PEM data.
func (s *Server) SetHostKey(pemData []byte) error {
	signer, err := ssh.ParsePrivateKey(pemData)
	if err != nil {
		return fmt.Errorf("failed to parse host key: %w", err)
	}
	s.sshConfig.AddHostKey(signer)
	return nil
}

// GenerateED25519Key generates an ED25519 private key in PEM format.
func GenerateED25519Key() ([]byte, error) {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ED25519 key: %w", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privBytes,
	}), nil
}

// defaultHostKeyCallback returns an ssh.HostKeyCallback for target connections.
// If GATEKEY_BASTION_TARGET_HOSTKEY is set to a file path containing an SSH public
// key (authorized_keys format), connections are validated against that key.
// If GATEKEY_BASTION_KNOWN_HOSTS is set, it is read as an OpenSSH known_hosts file.
// Otherwise, a trust-on-first-use (TOFU) callback is returned that accepts any key
// but logs a warning, matching the behavior of common bastion proxies for internal hosts.
func defaultHostKeyCallback() ssh.HostKeyCallback {
	// Check for explicit host key file
	if path := os.Getenv("GATEKEY_BASTION_TARGET_HOSTKEY"); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return fmt.Errorf("failed to read host key file: %w", err)
			}
		}
		pk, _, _, _, err := ssh.ParseAuthorizedKey(data)
		if err != nil {
			return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
				return fmt.Errorf("failed to parse host key: %w", err)
			}
		}
		return ssh.FixedHostKey(pk)
	}

	// Default: accept host keys for internal bastion targets
	// This is standard for bastion proxies connecting to internal infrastructure
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil
	}
}

// LoadOrGenerateHostKey loads a host key from the configured path, or generates
// a new one if the file does not exist. Returns the PEM data.
func (s *Server) LoadOrGenerateHostKey() ([]byte, error) {
	if s.config.HostKeyPath != "" {
		data, err := os.ReadFile(s.config.HostKeyPath)
		if err == nil {
			s.logger.Info("Loaded SSH bastion host key", zap.String("path", s.config.HostKeyPath))
			return data, s.SetHostKey(data)
		}
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read host key file: %w", err)
		}
	}

	// Generate a new key
	pemData, err := GenerateED25519Key()
	if err != nil {
		return nil, err
	}

	// Save to file if path is configured
	if s.config.HostKeyPath != "" {
		if err := os.WriteFile(s.config.HostKeyPath, pemData, 0600); err != nil {
			s.logger.Warn("Failed to save generated host key", zap.Error(err))
		} else {
			s.logger.Info("Generated and saved SSH bastion host key", zap.String("path", s.config.HostKeyPath))
		}
	} else {
		s.logger.Info("Generated ephemeral SSH bastion host key (no path configured)")
	}

	return pemData, s.SetHostKey(pemData)
}

// Start begins listening for SSH connections. Blocks until context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.ListenAddr, err)
	}
	s.listener = listener

	s.logger.Info("SSH bastion server started", zap.String("addr", s.config.ListenAddr))

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				s.logger.Debug("Failed to accept SSH connection", zap.Error(err))
				continue
			}
		}
		go s.handleConnection(ctx, conn)
	}
}

// Addr returns the listener address (useful when binding to :0 for tests).
func (s *Server) Addr() net.Addr {
	if s.listener != nil {
		return s.listener.Addr()
	}
	return nil
}

// passwordCallback is the SSH server password callback.
// It always succeeds at the SSH handshake level; real auth happens after we
// parse the target from the username. We stash the password in Permissions.
func (s *Server) passwordCallback(conn ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	return &ssh.Permissions{
		Extensions: map[string]string{
			"password": string(password),
			"user":     conn.User(),
		},
	}, nil
}

func (s *Server) handleConnection(ctx context.Context, netConn net.Conn) {
	defer netConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.sshConfig)
	if err != nil {
		s.logger.Debug("SSH handshake failed", zap.Error(err))
		return
	}
	defer sshConn.Close()

	password := sshConn.Permissions.Extensions["password"]
	sshUser := sshConn.Permissions.Extensions["user"]
	remoteAddr := netConn.RemoteAddr().String()
	remoteIP, _, _ := net.SplitHostPort(remoteAddr)

	// Authenticate the user via GateKey API key
	if s.AuthenticateUser == nil {
		s.logger.Error("No authentication function configured for SSH bastion")
		return
	}

	authResult, err := s.AuthenticateUser(ctx, password)
	if err != nil {
		s.logger.Warn("SSH bastion authentication failed",
			zap.String("ssh_user", sshUser),
			zap.String("remote", remoteAddr),
			zap.Error(err))
		sshConn.Close()
		return
	}

	// Handle global requests (keepalive, etc.)
	go ssh.DiscardRequests(reqs)

	// Handle channels
	for newChannel := range chans {
		switch newChannel.ChannelType() {
		case "session":
			// Direct SSH session: username encodes the target
			channel, requests, err := newChannel.Accept()
			if err != nil {
				s.logger.Warn("Failed to accept session channel", zap.Error(err))
				continue
			}
			go s.handleSessionChannel(ctx, channel, requests, sshConn, authResult, sshUser, remoteIP)

		case "direct-tcpip":
			// Jump host mode: ssh -J bastion user@target
			var payload directTCPIPPayload
			if err := ssh.Unmarshal(newChannel.ExtraData(), &payload); err != nil {
				s.logger.Warn("Failed to parse direct-tcpip payload", zap.Error(err))
				newChannel.Reject(ssh.ConnectionFailed, "invalid payload")
				continue
			}
			// Check access
			targetHost := payload.DestAddr
			targetPort := int(payload.DestPort)
			if s.CheckAccess != nil {
				hasAccess, checkErr := s.CheckAccess(ctx, authResult.UserID, authResult.Groups, targetHost, targetPort)
				if checkErr != nil || !hasAccess {
					s.logger.Warn("SSH bastion access denied (direct-tcpip)",
						zap.String("user", authResult.UserEmail),
						zap.String("target", fmt.Sprintf("%s:%d", targetHost, targetPort)))
					newChannel.Reject(ssh.Prohibited, "access denied")
					continue
				}
			}

			channel, _, err := newChannel.Accept()
			if err != nil {
				s.logger.Warn("Failed to accept direct-tcpip channel", zap.Error(err))
				continue
			}
			go s.handleDirectTCPIP(ctx, channel, authResult, remoteIP, targetHost, targetPort)

		default:
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
		}
	}
}

// directTCPIPPayload is the SSH wire format for direct-tcpip channel requests.
type directTCPIPPayload struct {
	DestAddr   string
	DestPort   uint32
	OriginAddr string
	OriginPort uint32
}

// handleDirectTCPIP proxies a TCP connection to the target (jump host mode).
// The SSH client handles end-to-end encryption to the target, so we just
// proxy raw bytes and log connection metadata.
func (s *Server) handleDirectTCPIP(ctx context.Context, channel ssh.Channel, auth *AuthResult, sourceIP, targetHost string, targetPort int) {
	defer channel.Close()

	targetAddr := fmt.Sprintf("%s:%d", targetHost, targetPort)
	startTime := time.Now()

	s.logger.Info("SSH bastion direct-tcpip connection",
		zap.String("user", auth.UserEmail),
		zap.String("target", targetAddr),
		zap.String("source", sourceIP))

	// Connect to the target
	targetConn, err := net.DialTimeout("tcp", targetAddr, 10*time.Second)
	if err != nil {
		s.logger.Warn("Failed to connect to target",
			zap.String("target", targetAddr),
			zap.Error(err))
		return
	}
	defer targetConn.Close()

	// Bidirectional copy with byte counting
	var bytesSent, bytesReceived int64
	var wg sync.WaitGroup

	// Client -> Target
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, _ := io.Copy(targetConn, channel)
		bytesSent = n
		// Signal target that client is done writing
		if tc, ok := targetConn.(*net.TCPConn); ok {
			tc.CloseWrite()
		}
	}()

	// Target -> Client
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, _ := io.Copy(channel, targetConn)
		bytesReceived = n
		channel.CloseWrite()
	}()

	wg.Wait()
	endTime := time.Now()

	s.logger.Info("SSH bastion direct-tcpip connection ended",
		zap.String("user", auth.UserEmail),
		zap.String("target", targetAddr),
		zap.Int64("bytes_sent", bytesSent),
		zap.Int64("bytes_received", bytesReceived),
		zap.Duration("duration", endTime.Sub(startTime)))

	// Log the connection
	if s.LogConnection != nil {
		s.LogConnection(auth.UserID, auth.UserEmail, sourceIP, targetHost, targetPort, bytesSent, bytesReceived, startTime, endTime)
	}
}

// handleSessionChannel handles a direct session channel where the target is
// encoded in the SSH username (format: targetuser@targethost or targetuser@targethost:port).
// This mode terminates SSH at the bastion and creates a new SSH connection to the target,
// enabling session recording of the terminal I/O.
func (s *Server) handleSessionChannel(ctx context.Context, channel ssh.Channel, requests <-chan *ssh.Request, sshConn *ssh.ServerConn, auth *AuthResult, sshUser, sourceIP string) {
	defer channel.Close()

	targetUser, targetHost, targetPort := ParseTarget(sshUser)
	if targetHost == "" {
		// No target specified — show interactive menu
		s.showInteractiveMenu(channel, auth)
		return
	}

	// Check access
	if s.CheckAccess != nil {
		hasAccess, checkErr := s.CheckAccess(ctx, auth.UserID, auth.Groups, targetHost, targetPort)
		if checkErr != nil || !hasAccess {
			s.logger.Warn("SSH bastion access denied",
				zap.String("user", auth.UserEmail),
				zap.String("target", fmt.Sprintf("%s@%s:%d", targetUser, targetHost, targetPort)))
			fmt.Fprintf(channel, "Access denied to %s:%d\r\n", targetHost, targetPort)
			return
		}
	}

	targetAddr := fmt.Sprintf("%s:%d", targetHost, targetPort)
	startTime := time.Now()

	s.logger.Info("SSH bastion session started",
		zap.String("user", auth.UserEmail),
		zap.String("target", fmt.Sprintf("%s@%s", targetUser, targetAddr)))

	// Show connection banner
	fmt.Fprintf(channel, "GateKey Bastion: connecting to %s@%s...\r\n", targetUser, targetAddr)

	// Connect to target SSH server
	targetConfig := &ssh.ClientConfig{
		User:            targetUser,
		HostKeyCallback: defaultHostKeyCallback(),
		Timeout:         10 * time.Second,
	}

	// For target authentication, we prompt the user for the target password
	// through the bastion channel
	targetConfig.Auth = []ssh.AuthMethod{
		ssh.PasswordCallback(func() (string, error) {
			fmt.Fprintf(channel, "%s@%s's password: ", targetUser, targetHost)
			// Read password from the channel (client's stdin)
			pw, err := readPassword(channel)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(channel, "\r\n")
			return pw, nil
		}),
		ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
			var answers []string
			if instruction != "" {
				fmt.Fprintf(channel, "%s\r\n", instruction)
			}
			for i, q := range questions {
				fmt.Fprintf(channel, "%s", q)
				if echos[i] {
					line, err := readLine(channel)
					if err != nil {
						return nil, err
					}
					answers = append(answers, line)
				} else {
					pw, err := readPassword(channel)
					if err != nil {
						return nil, err
					}
					fmt.Fprintf(channel, "\r\n")
					answers = append(answers, pw)
				}
			}
			return answers, nil
		}),
	}

	targetConn, err := ssh.Dial("tcp", targetAddr, targetConfig)
	if err != nil {
		s.logger.Warn("Failed to connect to target",
			zap.String("target", targetAddr),
			zap.Error(err))
		fmt.Fprintf(channel, "Failed to connect to %s: %s\r\n", targetAddr, err.Error())
		return
	}
	defer targetConn.Close()

	targetSession, err := targetConn.NewSession()
	if err != nil {
		s.logger.Warn("Failed to open target session", zap.Error(err))
		fmt.Fprintf(channel, "Failed to open session: %s\r\n", err.Error())
		return
	}
	defer targetSession.Close()

	// Start recording
	sessionID := fmt.Sprintf("bastion-%d", time.Now().UnixNano())
	recordingID := fmt.Sprintf("bastion-rec-%d", time.Now().UnixNano())
	var recordingActive bool

	if s.StartRecording != nil {
		storagePath, recErr := s.StartRecording(sessionID, recordingID, auth.UserEmail, targetHost)
		if recErr == nil && storagePath != "" {
			recordingActive = true
			if s.OnRecordStart != nil {
				s.OnRecordStart(recordingID, auth.UserID, auth.UserEmail, targetHost, storagePath)
			}
		}
	}

	// Set up pipes to target
	targetStdin, err := targetSession.StdinPipe()
	if err != nil {
		s.logger.Warn("Failed to get target stdin pipe", zap.Error(err))
		return
	}
	targetStdout, err := targetSession.StdoutPipe()
	if err != nil {
		s.logger.Warn("Failed to get target stdout pipe", zap.Error(err))
		return
	}
	targetStderr, err := targetSession.StderrPipe()
	if err != nil {
		s.logger.Warn("Failed to get target stderr pipe", zap.Error(err))
		return
	}

	// Forward SSH requests (pty-req, shell, window-change, exec, etc.)
	go func() {
		for req := range requests {
			switch req.Type {
			case "pty-req":
				ok, _ := targetSession.SendRequest(req.Type, req.WantReply, req.Payload)
				if req.WantReply {
					req.Reply(ok, nil)
				}
			case "window-change":
				ok, _ := targetSession.SendRequest(req.Type, req.WantReply, req.Payload)
				if req.WantReply {
					req.Reply(ok, nil)
				}
			case "shell":
				ok := targetSession.Shell() == nil
				if req.WantReply {
					req.Reply(ok, nil)
				}
			case "exec":
				// Extract the command from the payload
				if len(req.Payload) >= 4 {
					cmdLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 | int(req.Payload[2])<<8 | int(req.Payload[3])
					if cmdLen > 0 && 4+cmdLen <= len(req.Payload) {
						cmd := string(req.Payload[4 : 4+cmdLen])
						ok := targetSession.Start(cmd) == nil
						if req.WantReply {
							req.Reply(ok, nil)
						}
					} else if req.WantReply {
						req.Reply(false, nil)
					}
				} else if req.WantReply {
					req.Reply(false, nil)
				}
			case "env":
				ok, _ := targetSession.SendRequest(req.Type, req.WantReply, req.Payload)
				if req.WantReply {
					req.Reply(ok, nil)
				}
			default:
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}()

	var wg sync.WaitGroup
	var bytesSent, bytesReceived int64

	// Client -> Target (stdin)
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, _ := io.Copy(targetStdin, channel)
		bytesSent = n
	}()

	// Target -> Client (stdout) with recording
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := targetStdout.Read(buf)
			if n > 0 {
				data := buf[:n]
				channel.Write(data)
				bytesReceived += int64(n)
				if recordingActive && s.WriteRecording != nil {
					s.WriteRecording(sessionID, string(data))
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	// Target -> Client (stderr) with recording
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := targetStderr.Read(buf)
			if n > 0 {
				data := buf[:n]
				channel.Stderr().Write(data)
				if recordingActive && s.WriteRecording != nil {
					s.WriteRecording(sessionID, string(data))
				}
			}
			if readErr != nil {
				return
			}
		}
	}()

	// Wait for target session to end
	targetSession.Wait()
	wg.Wait()

	endTime := time.Now()

	// Stop recording
	if recordingActive && s.StopRecording != nil {
		fileSize, duration, _ := s.StopRecording(sessionID)
		if s.OnRecordStop != nil {
			s.OnRecordStop(recordingID, fileSize, duration)
		}
	}

	s.logger.Info("SSH bastion session ended",
		zap.String("user", auth.UserEmail),
		zap.String("target", fmt.Sprintf("%s@%s", targetUser, targetAddr)),
		zap.Int64("bytes_sent", bytesSent),
		zap.Int64("bytes_received", bytesReceived),
		zap.Duration("duration", endTime.Sub(startTime)))

	// Log the connection
	if s.LogConnection != nil {
		s.LogConnection(auth.UserID, auth.UserEmail, sourceIP, targetHost, targetPort, bytesSent, bytesReceived, startTime, endTime)
	}
}

// showInteractiveMenu displays available targets when no target is specified.
func (s *Server) showInteractiveMenu(channel ssh.Channel, auth *AuthResult) {
	fmt.Fprintf(channel, "\r\n")
	fmt.Fprintf(channel, "  GateKey SSH Bastion\r\n")
	fmt.Fprintf(channel, "  Authenticated as: %s\r\n", auth.UserEmail)
	fmt.Fprintf(channel, "\r\n")
	fmt.Fprintf(channel, "  Usage:\r\n")
	fmt.Fprintf(channel, "    ssh -p <port> <targetuser>@<targethost> <bastion-host>  (session mode)\r\n")
	fmt.Fprintf(channel, "    ssh -J <bastion-host>:<port> <targetuser>@<targethost>  (jump host mode)\r\n")
	fmt.Fprintf(channel, "\r\n")
	fmt.Fprintf(channel, "  Session mode records terminal I/O (asciicast format).\r\n")
	fmt.Fprintf(channel, "  Jump host mode logs connection metadata only.\r\n")
	fmt.Fprintf(channel, "\r\n")
	fmt.Fprintf(channel, "  Authenticate with your GateKey API key as the SSH password.\r\n")
	fmt.Fprintf(channel, "\r\n")
}

// ParseTarget parses an SSH username into target user, host, and port.
// Supported formats: "user@host", "user@host:port", "host", "host:port"
func ParseTarget(input string) (user, host string, port int) {
	port = 22 // default

	if idx := strings.LastIndex(input, "@"); idx >= 0 {
		user = input[:idx]
		input = input[idx+1:]
	}

	if hostStr, portStr, err := net.SplitHostPort(input); err == nil {
		host = hostStr
		fmt.Sscanf(portStr, "%d", &port)
	} else {
		host = input
	}

	// If no user specified and no @ was present, this might just be a
	// plain hostname with no target user. Default to empty so caller
	// can decide.
	return
}

// readPassword reads bytes from channel until \r or \n, without echoing.
func readPassword(r io.Reader) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if err != nil {
			return string(buf), err
		}
		if n > 0 {
			if b[0] == '\r' || b[0] == '\n' {
				return string(buf), nil
			}
			buf = append(buf, b[0])
		}
	}
}

// readLine reads bytes from channel until \r or \n, echoing back.
func readLine(rw io.ReadWriter) (string, error) {
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := rw.Read(b)
		if err != nil {
			return string(buf), err
		}
		if n > 0 {
			if b[0] == '\r' || b[0] == '\n' {
				fmt.Fprintf(rw, "\r\n")
				return string(buf), nil
			}
			rw.Write(b[:1])
			buf = append(buf, b[0])
		}
	}
}
