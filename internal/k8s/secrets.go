// Package k8s provides Kubernetes integration utilities.
package k8s

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// AdminPasswordSecretName is the name of the secret storing the admin password
	AdminPasswordSecretName = "gatekey-admin-init"
	// AdminPasswordKey is the key in the secret for the admin password
	AdminPasswordKey = "admin-password"
)

// SecretManager handles Kubernetes secret operations
type SecretManager struct {
	client    kubernetes.Interface
	namespace string
}

// NewSecretManager creates a new SecretManager using in-cluster config
// Returns nil if not running in Kubernetes
func NewSecretManager() (*SecretManager, error) {
	// Check if we're running in Kubernetes by looking for the service account
	if _, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token"); os.IsNotExist(err) {
		return nil, nil // Not in Kubernetes
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get namespace from the mounted service account
	namespace, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return nil, fmt.Errorf("failed to read namespace: %w", err)
	}

	return &SecretManager{
		client:    clientset,
		namespace: string(namespace),
	}, nil
}

// SaveAdminPassword saves the admin password to a Kubernetes secret
// Creates the secret if it doesn't exist, updates if it does
func (m *SecretManager) SaveAdminPassword(ctx context.Context, password string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AdminPasswordSecretName,
			Namespace: m.namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":      "gatekey",
				"app.kubernetes.io/component": "admin-init",
				"app.kubernetes.io/managed-by": "gatekey-server",
			},
			Annotations: map[string]string{
				"description": "Initial admin password for GateKey. Delete this secret after changing the password.",
			},
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			AdminPasswordKey: password,
			"username":       "admin",
			"note":           "Please change this password after first login and delete this secret.",
		},
	}

	// Try to create the secret
	_, err := m.client.CoreV1().Secrets(m.namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		if errors.IsAlreadyExists(err) {
			// Secret already exists, update it
			_, err = m.client.CoreV1().Secrets(m.namespace).Update(ctx, secret, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("failed to update admin password secret: %w", err)
			}
			return nil
		}
		return fmt.Errorf("failed to create admin password secret: %w", err)
	}

	return nil
}

// GetAdminPassword retrieves the admin password from the Kubernetes secret
// Returns empty string if the secret doesn't exist
func (m *SecretManager) GetAdminPassword(ctx context.Context) (string, error) {
	secret, err := m.client.CoreV1().Secrets(m.namespace).Get(ctx, AdminPasswordSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to get admin password secret: %w", err)
	}

	password, ok := secret.Data[AdminPasswordKey]
	if !ok {
		return "", nil
	}

	return string(password), nil
}

// DeleteAdminPasswordSecret deletes the admin password secret
func (m *SecretManager) DeleteAdminPasswordSecret(ctx context.Context) error {
	err := m.client.CoreV1().Secrets(m.namespace).Delete(ctx, AdminPasswordSecretName, metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete admin password secret: %w", err)
	}
	return nil
}

// AdminPasswordSecretExists checks if the admin password secret exists
func (m *SecretManager) AdminPasswordSecretExists(ctx context.Context) (bool, error) {
	_, err := m.client.CoreV1().Secrets(m.namespace).Get(ctx, AdminPasswordSecretName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check admin password secret: %w", err)
	}
	return true, nil
}
