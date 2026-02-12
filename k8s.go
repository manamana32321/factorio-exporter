package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// K8sClient provides in-cluster Kubernetes API access.
type K8sClient struct {
	namespace string
}

func NewK8sClient(namespace string) *K8sClient {
	return &K8sClient{namespace: namespace}
}

func (k *K8sClient) FindPod(ctx context.Context, labelSelector string) (string, error) {
	client, token, err := k.httpClient()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods?labelSelector=%s&limit=1",
		k.apiBase(), k.namespace, labelSelector)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("list pods: %s %s", resp.Status, string(body))
	}

	var result struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Items) == 0 {
		return "", fmt.Errorf("no pods found with label %s", labelSelector)
	}
	return result.Items[0].Metadata.Name, nil
}

func (k *K8sClient) StreamLogs(ctx context.Context, podName string) (io.ReadCloser, error) {
	client, token, err := k.httpClient()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/v1/namespaces/%s/pods/%s/log?follow=true&sinceSeconds=10&timestamps=false",
		k.apiBase(), k.namespace, podName)

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	client.Timeout = 0

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("stream logs: %s %s", resp.Status, string(body))
	}

	return resp.Body, nil
}

func (k *K8sClient) httpClient() (*http.Client, string, error) {
	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, "", fmt.Errorf("read sa token: %w", err)
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}, string(token), nil
}

func (k *K8sClient) apiBase() string {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return "https://kubernetes.default.svc"
	}
	return fmt.Sprintf("https://%s:%s", host, port)
}
