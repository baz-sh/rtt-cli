package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
)

const baseURL = "https://api.rtt.io/api/v1/json"

type Client struct {
	httpClient *http.Client
	auth       string
}

type Departure struct {
	BookedDepartureTime string
	DeparturePlatform   string
	Platform            string
	ArrivingAt          string
	Duration            string
	Leaving             string
	Service             string
}

type searchResponse struct {
	Services []struct {
		ServiceUID     string `json:"serviceUid"`
		IsPassenger    bool   `json:"isPassenger"`
		LocationDetail struct {
			GBTTBookedDeparture string `json:"gbttBookedDeparture"`
			Platform            string `json:"platform"`
		} `json:"locationDetail"`
	} `json:"services"`
}

type serviceResponse struct {
	AtocName  string `json:"atocName"`
	Locations []struct {
		CRS               string `json:"crs"`
		GBTTBookedArrival string `json:"gbttBookedArrival"`
		Platform          string `json:"platform"`
	} `json:"locations"`
}

func NewClient(username, password string) *Client {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		auth:       "Basic " + auth,
	}
}

func (c *Client) GetDepartures(from, to string) ([]Departure, error) {
	if from == "" {
		return []Departure{}, nil
	}

	now := time.Now()
	dateStr := fmt.Sprintf("%d/%02d/%02d", now.Year(), now.Month(), now.Day())

	// Query time windows from now until end of day (2-hour steps to avoid gaps)
	var timeWindows []string
	for h := now.Hour(); h < 24; h += 2 {
		timeWindows = append(timeWindows, fmt.Sprintf("%02d00", h))
	}

	type searchResult struct {
		resp *searchResponse
		err  error
	}
	results := make([]searchResult, len(timeWindows))

	var wg sync.WaitGroup
	for i, tw := range timeWindows {
		wg.Add(1)
		go func(idx int, timeWindow string) {
			defer wg.Done()
			searchURL := fmt.Sprintf("%s/search/%s/to/%s/%s/%s",
				baseURL,
				url.PathEscape(strings.ToUpper(from)),
				url.PathEscape(strings.ToUpper(to)),
				dateStr,
				timeWindow)
			resp, err := c.fetchSearchResults(searchURL)
			results[idx] = searchResult{resp: resp, err: err}
		}(i, tw)
	}
	wg.Wait()

	// Deduplicate services by UID
	seen := make(map[string]bool)
	type serviceInfo struct {
		uid                 string
		bookedDepartureTime string
		platform            string
	}
	var allServices []serviceInfo

	for _, result := range results {
		if result.err != nil || result.resp == nil {
			continue
		}
		for _, svc := range result.resp.Services {
			if !svc.IsPassenger || seen[svc.ServiceUID] {
				continue
			}
			seen[svc.ServiceUID] = true
			allServices = append(allServices, serviceInfo{
				uid:                 svc.ServiceUID,
				bookedDepartureTime: svc.LocationDetail.GBTTBookedDeparture,
				platform:            svc.LocationDetail.Platform,
			})
		}
	}

	if len(allServices) == 0 {
		return []Departure{}, nil
	}

	// Get detailed info for each service concurrently
	type departureResult struct {
		departure *Departure
	}
	depResults := make([]departureResult, len(allServices))

	for i, svc := range allServices {
		wg.Add(1)
		go func(idx int, s serviceInfo) {
			defer wg.Done()
			serviceURL := fmt.Sprintf("%s/service/%s/%s",
				baseURL,
				s.uid,
				dateStr)

			req, err := http.NewRequest("GET", serviceURL, nil)
			if err != nil {
				return
			}
			req.Header.Set("Authorization", c.auth)

			resp, err := c.httpClient.Do(req)
			if err != nil {
				return
			}

			var serviceResp serviceResponse
			if err := json.NewDecoder(resp.Body).Decode(&serviceResp); err != nil {
				resp.Body.Close()
				return
			}
			resp.Body.Close()

			depResults[idx].departure = c.buildDeparture(
				&serviceResp,
				strings.ToUpper(to),
				s.bookedDepartureTime,
				s.platform,
			)
		}(i, svc)
	}
	wg.Wait()

	// Collect results, filtering out departed trains
	currentTime := getCurrentTime()
	var departures []Departure
	for _, dr := range depResults {
		if dr.departure != nil && dr.departure.BookedDepartureTime >= currentTime {
			departures = append(departures, *dr.departure)
		}
	}

	// Sort by departure time
	sort.Slice(departures, func(i, j int) bool {
		return departures[i].BookedDepartureTime < departures[j].BookedDepartureTime
	})

	return departures, nil
}

func (c *Client) fetchSearchResults(searchURL string) (*searchResponse, error) {
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.auth)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch departures: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var searchResp searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &searchResp, nil
}

func (c *Client) buildDeparture(
	serviceResp *serviceResponse,
	to string,
	bookedDepartureTime string,
	departurePlatform string,
) *Departure {
	// Find arrival data
	var arrivalData *struct {
		CRS               string `json:"crs"`
		GBTTBookedArrival string `json:"gbttBookedArrival"`
		Platform          string `json:"platform"`
	}

	for i := range serviceResp.Locations {
		if serviceResp.Locations[i].CRS == to {
			arrivalData = &serviceResp.Locations[i]
			break
		}
	}

	if arrivalData == nil {
		return nil
	}

	duration := getDurationBetween(bookedDepartureTime, arrivalData.GBTTBookedArrival)
	leaving := getDurationBetween(getCurrentTime(), bookedDepartureTime)

	// Default to platform "1" if no platform is specified
	depPlatform := departurePlatform
	if depPlatform == "" {
		depPlatform = "1"
	}

	arrPlatform := arrivalData.Platform
	if arrPlatform == "" {
		arrPlatform = "1"
	}

	return &Departure{
		BookedDepartureTime: bookedDepartureTime,
		DeparturePlatform:   depPlatform,
		Platform:            arrPlatform,
		ArrivingAt:          arrivalData.GBTTBookedArrival,
		Duration:            duration,
		Leaving:             leaving,
		Service:             serviceResp.AtocName,
	}
}

func getDurationBetween(startTime, endTime string) string {
	if len(startTime) < 4 || len(endTime) < 4 {
		return "N/A"
	}

	startHours := parseInt(startTime[0:2])
	startMinutes := parseInt(startTime[2:4])
	startTotalMinutes := startHours*60 + startMinutes

	endHours := parseInt(endTime[0:2])
	endMinutes := parseInt(endTime[2:4])
	endTotalMinutes := endHours*60 + endMinutes

	// Handle day rollover
	if endTotalMinutes < startTotalMinutes {
		endTotalMinutes += 1440
	}

	durationMinutes := endTotalMinutes - startTotalMinutes
	durationHours := durationMinutes / 60
	durationRemainingMinutes := durationMinutes % 60

	if durationHours == 0 {
		return fmt.Sprintf("%dmin", durationRemainingMinutes)
	}
	return fmt.Sprintf("%dhr %dmin", durationHours, durationRemainingMinutes)
}

func getCurrentTime() string {
	now := time.Now()
	return fmt.Sprintf("%02d%02d", now.Hour(), now.Minute())
}

func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}
