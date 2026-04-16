package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://data.rtt.io"

type Client struct {
	httpClient   *http.Client
	refreshToken string
	accessToken  string
	tokenExpiry  time.Time
}

type Departure struct {
	BookedDepartureTime string
	DeparturePlatform   string
	Platform            string
	ArrivingAt          string
	Duration            string
	Leaving             string
	Service             string
	departureTime       time.Time // parsed, used for filtering/sorting
}

// v2 API response types

type locationResponse struct {
	Services []locationService `json:"services"`
}

type locationService struct {
	ScheduleMetadata struct {
		UniqueIdentity     string `json:"uniqueIdentity"`
		Operator           struct {
			Name string `json:"name"`
		} `json:"operator"`
		InPassengerService bool `json:"inPassengerService"`
	} `json:"scheduleMetadata"`
	TemporalData struct {
		Departure *temporalData `json:"departure"`
	} `json:"temporalData"`
	LocationMetadata struct {
		Platform *plannedActualData `json:"platform"`
	} `json:"locationMetadata"`
}

type serviceResponse struct {
	Service struct {
		ScheduleMetadata struct {
			UniqueIdentity string `json:"uniqueIdentity"`
			Operator       struct {
				Name string `json:"name"`
			} `json:"operator"`
		} `json:"scheduleMetadata"`
		Locations []serviceLocation `json:"locations"`
	} `json:"service"`
}

type serviceLocation struct {
	Location struct {
		ShortCodes []string `json:"shortCodes"`
	} `json:"location"`
	TemporalData struct {
		Arrival   *temporalData `json:"arrival"`
		Departure *temporalData `json:"departure"`
	} `json:"temporalData"`
	LocationMetadata struct {
		Platform *plannedActualData `json:"platform"`
	} `json:"locationMetadata"`
}

type temporalData struct {
	ScheduleAdvertised string `json:"scheduleAdvertised"`
	RealtimeForecast   string `json:"realtimeForecast"`
	RealtimeActual     string `json:"realtimeActual"`
	IsCancelled        bool   `json:"isCancelled"`
}

type plannedActualData struct {
	Planned  string `json:"planned"`
	Forecast string `json:"forecast"`
	Actual   string `json:"actual"`
}

type serviceInfo struct {
	uniqueIdentity      string
	bookedDepartureTime string
	platform            string
	operator            string
}

type accessTokenResponse struct {
	Token      string `json:"token"`
	ValidUntil string `json:"validUntil"`
}

func NewClient(refreshToken string) *Client {
	return &Client{
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		refreshToken: refreshToken,
	}
}

// ensureAccessToken exchanges the refresh token for a short-life access token if needed.
func (c *Client) ensureAccessToken() error {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return nil
	}

	req, err := http.NewRequest("GET", baseURL+"/api/get_access_token", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.refreshToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp accessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	c.accessToken = tokenResp.Token
	if expiry, err := time.Parse(time.RFC3339, tokenResp.ValidUntil); err == nil {
		// Refresh a bit early to avoid edge cases
		c.tokenExpiry = expiry.Add(-30 * time.Second)
	} else {
		// If we can't parse expiry, refresh after 5 minutes
		c.tokenExpiry = time.Now().Add(5 * time.Minute)
	}

	return nil
}

func (c *Client) GetDepartures(from, to string) ([]Departure, error) {
	now := time.Now()
	from = strings.ToUpper(from)
	to = strings.ToUpper(to)

	services, err := c.fetchServices(from, to, now)
	if err != nil {
		return nil, err
	}

	departures := c.fetchDepartureDetails(services, to)

	// Filter out departed trains and sort by departure time
	now = time.Now()
	var result []Departure
	for _, dep := range departures {
		if !dep.departureTime.Before(now) {
			result = append(result, dep)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].departureTime.Before(result[j].departureTime)
	})

	return result, nil
}

// fetchServices returns passenger services from a station filtered by destination.
func (c *Client) fetchServices(from, to string, now time.Time) ([]serviceInfo, error) {
	params := url.Values{}
	params.Set("code", from)
	params.Set("filterTo", to)
	params.Set("timeFrom", now.Format("2006-01-02T15:04:05"))
	params.Set("timeWindow", "1439")
	locationURL := fmt.Sprintf("%s/gb-nr/location?%s", baseURL, params.Encode())

	raw, err := c.fetchJSON(locationURL)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, nil
	}

	var resp locationResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, fmt.Errorf("failed to decode location response: %w", err)
	}

	var services []serviceInfo
	for _, svc := range resp.Services {
		if !svc.ScheduleMetadata.InPassengerService {
			continue
		}

		depTime := ""
		if svc.TemporalData.Departure != nil {
			depTime = svc.TemporalData.Departure.ScheduleAdvertised
		}

		platform := ""
		if svc.LocationMetadata.Platform != nil {
			platform = bestPlatform(svc.LocationMetadata.Platform)
		}

		services = append(services, serviceInfo{
			uniqueIdentity:      svc.ScheduleMetadata.UniqueIdentity,
			bookedDepartureTime: depTime,
			platform:            platform,
			operator:            svc.ScheduleMetadata.Operator.Name,
		})
	}

	return services, nil
}

// fetchDepartureDetails fetches full service details concurrently and builds departures.
func (c *Client) fetchDepartureDetails(services []serviceInfo, to string) []Departure {
	results := make([]*Departure, len(services))
	sem := make(chan struct{}, 3)
	var wg sync.WaitGroup

	for i, svc := range services {
		wg.Add(1)
		go func(idx int, s serviceInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// uniqueIdentity is "gb-nr:IDENTITY:DATE"
			parts := strings.SplitN(s.uniqueIdentity, ":", 3)
			if len(parts) != 3 {
				return
			}
			params := url.Values{}
			params.Set("identity", parts[1])
			params.Set("departureDate", parts[2])

			raw, err := c.fetchJSON(fmt.Sprintf("%s/gb-nr/service?%s", baseURL, params.Encode()))
			if err != nil || raw == nil {
				return
			}

			var svcResp serviceResponse
			if err := json.Unmarshal(raw, &svcResp); err != nil {
				return
			}

			results[idx] = buildDeparture(&svcResp, to, s)
		}(i, svc)
	}
	wg.Wait()

	var departures []Departure
	for _, dep := range results {
		if dep != nil {
			departures = append(departures, *dep)
		}
	}
	return departures
}

// fetchJSON makes an authenticated GET request and returns the raw response body.
// Returns nil, nil for 204 (no content) responses. Retries on rate limiting.
func (c *Client) fetchJSON(rawURL string) (json.RawMessage, error) {
	if err := c.ensureAccessToken(); err != nil {
		return nil, err
	}

	for attempt := range 3 {
		raw, err := c.doGet(rawURL)
		if err == errRateLimited {
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		return raw, err
	}

	return nil, fmt.Errorf("API rate limited after retries")
}

var errRateLimited = fmt.Errorf("rate limited")

func (c *Client) doGet(rawURL string) (json.RawMessage, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, errRateLimited
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return raw, nil
}

// bestPlatform returns the most up-to-date platform info available.
func bestPlatform(p *plannedActualData) string {
	if p.Actual != "" {
		return p.Actual
	}
	if p.Forecast != "" {
		return p.Forecast
	}
	return p.Planned
}

func buildDeparture(svcResp *serviceResponse, to string, info serviceInfo) *Departure {
	depTime := parseAPITime(info.bookedDepartureTime)
	if depTime.IsZero() {
		return nil
	}

	arrLoc := findLocation(svcResp.Service.Locations, to)
	if arrLoc == nil {
		return nil
	}

	var arrTime time.Time
	if arrLoc.TemporalData.Arrival != nil {
		arrTime = parseAPITime(arrLoc.TemporalData.Arrival.ScheduleAdvertised)
	}

	arrPlatform := ""
	if arrLoc.LocationMetadata.Platform != nil {
		arrPlatform = bestPlatform(arrLoc.LocationMetadata.Platform)
	}

	depPlatform := info.platform
	if depPlatform == "" {
		depPlatform = "1"
	}
	if arrPlatform == "" {
		arrPlatform = "1"
	}

	return &Departure{
		BookedDepartureTime: depTime.Format("15:04"),
		DeparturePlatform:   depPlatform,
		Platform:            arrPlatform,
		ArrivingAt:          arrTime.Format("15:04"),
		Duration:            formatDuration(depTime, arrTime),
		Leaving:             formatDuration(time.Now(), depTime),
		Service:             info.operator,
		departureTime:       depTime,
	}
}

func findLocation(locations []serviceLocation, code string) *serviceLocation {
	for i := range locations {
		for _, sc := range locations[i].Location.ShortCodes {
			if strings.EqualFold(sc, code) {
				return &locations[i]
			}
		}
	}
	return nil
}

// parseAPITime parses an ISO 8601 datetime string from the API.
// Times without a timezone offset are treated as local time.
func parseAPITime(s string) time.Time {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04:05", s, time.Local); err == nil {
		return t
	}
	return time.Time{}
}

func formatDuration(from, to time.Time) string {
	if from.IsZero() || to.IsZero() {
		return "N/A"
	}

	d := to.Sub(from)
	if d < 0 {
		return "N/A"
	}

	totalMinutes := int(d.Minutes())
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	if hours == 0 {
		return fmt.Sprintf("%dmin", minutes)
	}
	return fmt.Sprintf("%dhr %dmin", hours, minutes)
}
