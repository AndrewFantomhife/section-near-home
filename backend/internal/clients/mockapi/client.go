package mockapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"section-near-home/backend/internal/domain"
	"section-near-home/backend/internal/logging"
)

const defaultRequestTimeout = 10 * time.Second

// Client является HTTP-клиентом сервиса mock-api (json-server).
// Он изолирует бизнес-логику от HTTP-контракта справочника:
// при замене источника данных изменится только этот пакет.
type Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// NewClient создаёт клиент mock-api с указанным базовым адресом.
func NewClient(baseURL string, logger *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: defaultRequestTimeout,
		},
		logger: logging.WithComponent(logger, "mockapi"),
	}
}

// ListSections возвращает полный список секций из справочника.
func (client *Client) ListSections(ctx context.Context) ([]domain.Section, error) {
	requestURL := client.baseURL + "/sections"

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create sections request: %w", err)
	}

	client.logger.DebugContext(ctx, "requesting sections", "url", requestURL)

	response, err := client.httpClient.Do(request)
	if err != nil {
		logging.ErrorWithCause(client.logger, "sections request failed", err, "url", requestURL)
		return nil, fmt.Errorf("execute sections request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d from %s", response.StatusCode, requestURL)
	}

	var sections []domain.Section
	if err := json.NewDecoder(response.Body).Decode(&sections); err != nil {
		return nil, fmt.Errorf("decode sections: %w", err)
	}

	client.logger.DebugContext(ctx, "sections loaded", "count", len(sections))
	return sections, nil
}

// GetSection возвращает одну секцию по её идентификатору.
func (client *Client) GetSection(ctx context.Context, sectionID int) (domain.Section, error) {
	requestURL := fmt.Sprintf("%s/sections/%d", client.baseURL, sectionID)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return domain.Section{}, fmt.Errorf("create section request: %w", err)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		logging.ErrorWithCause(client.logger, "section request failed", err, "url", requestURL)
		return domain.Section{}, fmt.Errorf("execute section request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return domain.Section{}, fmt.Errorf("unexpected status %d from %s", response.StatusCode, requestURL)
	}

	var section domain.Section
	if err := json.NewDecoder(response.Body).Decode(&section); err != nil {
		return domain.Section{}, fmt.Errorf("decode section: %w", err)
	}

	return section, nil
}
