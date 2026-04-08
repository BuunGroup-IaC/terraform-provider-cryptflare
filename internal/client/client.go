// Copyright (c) 2026 Buun Group
// SPDX-License-Identifier: MPL-2.0

// Package client provides an HTTP client for the CryptFlare API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.cryptflare.com"

// Client wraps the CryptFlare REST API.
type Client struct {
	baseURL    string
	apiToken   string
	orgID      string
	httpClient *http.Client
}

// New creates a CryptFlare API client.
func New(baseURL, apiToken, orgID string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL:  baseURL,
		apiToken: apiToken,
		orgID:    orgID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OrgID returns the configured organisation ID.
func (c *Client) OrgID() string {
	return c.orgID
}

// APIError represents an error response from the CryptFlare API.
type APIError struct {
	StatusCode int
	Code       string `json:"error"`
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s (HTTP %d)", e.Code, e.Message, e.StatusCode)
}

// IsNotFound returns true if the error is a 404.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

func (c *Client) do(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.Unmarshal(respBody, apiErr)
		if apiErr.Message == "" {
			apiErr.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return apiErr
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}

// --- Workspace ---

// Workspace represents a CryptFlare workspace.
type Workspace struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisation_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	CreatedAt      string `json:"created_at"`
}

// CreateWorkspaceInput is the request body for creating a workspace.
type CreateWorkspaceInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ListWorkspaces returns all workspaces in the organisation.
func (c *Client) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	var resp struct {
		Data []Workspace `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/organisations/%s/workspaces", c.orgID), nil, &resp)
	return resp.Data, err
}

// GetWorkspace returns a workspace by ID or slug.
func (c *Client) GetWorkspace(ctx context.Context, idOrSlug string) (*Workspace, error) {
	var resp struct {
		Data Workspace `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/organisations/%s/workspaces/%s", c.orgID, idOrSlug), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreateWorkspace creates a new workspace.
func (c *Client) CreateWorkspace(ctx context.Context, input CreateWorkspaceInput) (*Workspace, error) {
	var resp struct {
		Data Workspace `json:"data"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/organisations/%s/workspaces", c.orgID), input, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeleteWorkspace permanently deletes a workspace.
func (c *Client) DeleteWorkspace(ctx context.Context, idOrSlug string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/v1/organisations/%s/workspaces/%s", c.orgID, idOrSlug), nil, nil)
}

// --- Environment ---

// Environment represents a CryptFlare environment.
type Environment struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	CreatedAt   string `json:"created_at"`
}

// CreateEnvironmentInput is the request body for creating an environment.
type CreateEnvironmentInput struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ListEnvironments returns all environments in a workspace.
func (c *Client) ListEnvironments(ctx context.Context, workspaceID string) ([]Environment, error) {
	var resp struct {
		Data []Environment `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("/v1/organisations/%s/workspaces/%s/environments", c.orgID, workspaceID), nil, &resp)
	return resp.Data, err
}

// CreateEnvironment creates a new environment.
func (c *Client) CreateEnvironment(ctx context.Context, workspaceID string, input CreateEnvironmentInput) (*Environment, error) {
	var resp struct {
		Data Environment `json:"data"`
	}
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("/v1/organisations/%s/workspaces/%s/environments", c.orgID, workspaceID), input, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// --- Secret ---

// Secret represents a CryptFlare secret (metadata only, no value).
type Secret struct {
	ID        string  `json:"id"`
	Key       string  `json:"key"`
	Version   int     `json:"version"`
	PodID     *string `json:"podId"`
	CreatedBy string  `json:"createdBy"`
	CreatedAt string  `json:"createdAt"`
	UpdatedAt string  `json:"updatedAt"`
}

// SecretValue represents a revealed secret.
type SecretValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// CreateSecretInput is the request body for creating a secret.
type CreateSecretInput struct {
	Key   string  `json:"key"`
	Value string  `json:"value"`
	PodID *string `json:"podId,omitempty"`
}

// CreateSecretResponse is returned after creating a secret.
type CreateSecretResponse struct {
	Key     string `json:"key"`
	Version int    `json:"version"`
}

func (c *Client) secretBasePath(workspaceID, envID string) string {
	return fmt.Sprintf("/v1/organisations/%s/workspaces/%s/environments/%s/secrets", c.orgID, workspaceID, envID)
}

// CreateSecret creates a new secret.
func (c *Client) CreateSecret(ctx context.Context, workspaceID, envID string, input CreateSecretInput) (*CreateSecretResponse, error) {
	var resp CreateSecretResponse
	err := c.do(ctx, http.MethodPost, c.secretBasePath(workspaceID, envID), input, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetSecret reveals a secret value.
func (c *Client) GetSecret(ctx context.Context, workspaceID, envID, key string) (*SecretValue, error) {
	var resp struct {
		Data SecretValue `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/%s", c.secretBasePath(workspaceID, envID), key), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// RotateSecret updates a secret to a new value.
func (c *Client) RotateSecret(ctx context.Context, workspaceID, envID, key, value string) (*CreateSecretResponse, error) {
	var resp CreateSecretResponse
	err := c.do(ctx, http.MethodPost, fmt.Sprintf("%s/%s/rotate", c.secretBasePath(workspaceID, envID), key), map[string]string{"value": value}, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteSecret permanently deletes a secret.
func (c *Client) DeleteSecret(ctx context.Context, workspaceID, envID, key string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", c.secretBasePath(workspaceID, envID), key), nil, nil)
}

// --- Pod ---

// Pod represents a CryptFlare pod (folder).
type Pod struct {
	ID            string  `json:"id"`
	EnvironmentID string  `json:"environmentId"`
	ParentID      *string `json:"parentId"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	Description   *string `json:"description"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

// CreatePodInput is the request body for creating a pod.
type CreatePodInput struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	ParentID    *string `json:"parentId,omitempty"`
	Description *string `json:"description,omitempty"`
}

// UpdatePodInput is the request body for updating a pod.
type UpdatePodInput struct {
	Name        *string `json:"name,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (c *Client) podBasePath(workspaceID, envID string) string {
	return fmt.Sprintf("/v1/organisations/%s/workspaces/%s/environments/%s/pods", c.orgID, workspaceID, envID)
}

// GetPod returns a pod by ID.
func (c *Client) GetPod(ctx context.Context, workspaceID, envID, podID string) (*Pod, error) {
	var resp struct {
		Data Pod `json:"data"`
	}
	err := c.do(ctx, http.MethodGet, fmt.Sprintf("%s/%s", c.podBasePath(workspaceID, envID), podID), nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// CreatePod creates a new pod.
func (c *Client) CreatePod(ctx context.Context, workspaceID, envID string, input CreatePodInput) (*Pod, error) {
	var resp struct {
		Data Pod `json:"data"`
	}
	err := c.do(ctx, http.MethodPost, c.podBasePath(workspaceID, envID), input, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// UpdatePod updates a pod.
func (c *Client) UpdatePod(ctx context.Context, workspaceID, envID, podID string, input UpdatePodInput) (*Pod, error) {
	var resp struct {
		Data Pod `json:"data"`
	}
	err := c.do(ctx, http.MethodPatch, fmt.Sprintf("%s/%s", c.podBasePath(workspaceID, envID), podID), input, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.Data, nil
}

// DeletePod deletes an empty pod.
func (c *Client) DeletePod(ctx context.Context, workspaceID, envID, podID string) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("%s/%s", c.podBasePath(workspaceID, envID), podID), nil, nil)
}
