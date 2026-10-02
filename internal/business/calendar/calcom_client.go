package calendar

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
)

type CalcomClient struct {
	apiKey      string
	eventTypeID string
	timezone    string
}

func NewCalcomClient(apiKey, eventTypeID, timezone string) *CalcomClient {
	return &CalcomClient{apiKey: apiKey, eventTypeID: eventTypeID, timezone: timezone}
}

// --- Busca de horários livres ---

type slotsResponse struct {
	Data map[string][]struct {
		Start string `json:"start"`
	} `json:"data"`
}

// GetAvailableSlots retorna os horários livres entre start e end, já
// respeitando a disponibilidade (seg/qua/sex 13h-17h) configurada no
// próprio Cal.com — não precisamos filtrar isso no nosso código.
func (c *CalcomClient) GetAvailableSlots(start, end time.Time) ([]time.Time, error) {
	query := url.Values{}
	query.Set("eventTypeId", c.eventTypeID)
	query.Set("start", start.Format(time.RFC3339))
	query.Set("end", end.Format(time.RFC3339))
	query.Set("timeZone", c.timezone)

	reqURL := "https://api.cal.com/v2/slots?" + query.Encode()

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao montar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("cal-api-version", "2024-09-04") // versão específica desse endpoint

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar Cal.com: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("cal.com retornou status %d ao buscar horários", resp.StatusCode)
	}

	var parsed slotsResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	// A resposta vem agrupada por data (mapa) — achatamos num slice único,
	// já convertendo cada string ISO pra time.Time de verdade.
	var slots []time.Time
	for _, daySlots := range parsed.Data {
		for _, s := range daySlots {
			t, err := time.Parse(time.RFC3339, s.Start)
			if err != nil {
				continue // ignora um slot malformado em vez de quebrar a lista inteira
			}
			slots = append(slots, t)
		}
	}

	sort.Slice(slots, func(i, j int) bool { return slots[i].Before(slots[j]) })

	return slots, nil
}

// --- Criação da reserva ---

type createBookingRequest struct {
	Start       string           `json:"start"`
	EventTypeID string           `json:"eventTypeId"`
	Attendee    bookingAttendee  `json:"attendee"`
}

type bookingAttendee struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	TimeZone string `json:"timeZone"`
}

type createBookingResponse struct {
	Status string `json:"status"`
	Data   struct {
		UID   string `json:"uid"`
		Start string `json:"start"`
	} `json:"data"`
}

// CreateBooking reserva o horário escolhido, vinculando o email/nome do
// cliente como participante. Retorna o UID da reserva no Cal.com — é esse
// UID que vamos guardar em consultations.external_booking_id.
func (c *CalcomClient) CreateBooking(start time.Time, attendeeName, attendeeEmail string) (string, error) {
	payload := createBookingRequest{
		Start:       start.Format(time.RFC3339),
		EventTypeID: c.eventTypeID,
		Attendee: bookingAttendee{
			Name:     attendeeName,
			Email:    attendeeEmail,
			TimeZone: c.timezone,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("erro ao serializar payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.cal.com/v2/bookings", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("erro ao montar requisição: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("cal-api-version", "2024-08-13") // versão específica desse endpoint (diferente da de slots!)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("erro ao chamar Cal.com: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("cal.com retornou status %d ao criar reserva", resp.StatusCode)
	}

	var parsed createBookingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("erro ao decodificar resposta: %w", err)
	}

	return parsed.Data.UID, nil
}