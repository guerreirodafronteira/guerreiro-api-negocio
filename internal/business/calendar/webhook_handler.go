package calendar

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/guerreirodafronteira/guerreiro-api-negocio/internal/business"
)

type CalcomWebhookHandler struct {
	repo           *business.Repository
	secret         string
	organizerEmail string
}

func NewCalcomWebhookHandler(repo *business.Repository, secret, organizerEmail string) *CalcomWebhookHandler {
	return &CalcomWebhookHandler{repo: repo, secret: secret, organizerEmail: organizerEmail}
}

type calcomPayload struct {
	TriggerEvent string `json:"triggerEvent"`
	Payload      struct {
		UID         string `json:"uid"`
		CancelledBy string `json:"cancelledBy"`
		StartTime   string `json:"startTime"`
		Attendees   []struct {
			Email  string `json:"email"`
			NoShow bool   `json:"noShow"`
		} `json:"attendees"`
	} `json:"payload"`
}

func (h *CalcomWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "erro ao ler corpo", http.StatusServiceUnavailable)
		return
	}

	// Verificação manual de assinatura: a Cal.com manda um HMAC-SHA256 do
	// corpo cru, usando o secret como chave — calculamos o mesmo hash aqui
	// e comparamos. hmac.Equal (em vez de == direto) evita "timing attacks",
	// onde alguém descobre o segredo medindo quanto tempo a comparação leva.
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))
	receivedSignature := r.Header.Get("x-cal-signature-256")

	if !hmac.Equal([]byte(expectedSignature), []byte(receivedSignature)) {
		log.Println("assinatura do webhook Cal.com inválida")
		http.Error(w, "assinatura inválida", http.StatusBadRequest)
		return
	}

	var event calcomPayload
	if err := json.Unmarshal(body, &event); err != nil {
		http.Error(w, "payload inválido", http.StatusBadRequest)
		return
	}

	bookingUID := event.Payload.UID
	ctx := r.Context()

	switch event.TriggerEvent {
	case "BOOKING_CANCELLED":
		if event.Payload.CancelledBy == h.organizerEmail {
			h.repo.UpdateConsultationStatus(ctx, bookingUID, "cancelado_pelo_consultor")
		} else {
			h.repo.UpdateConsultationStatus(ctx, bookingUID, "cancelado_pelo_cliente")
		}

	case "BOOKING_RESCHEDULED":
		startTime, err := time.Parse(time.RFC3339, event.Payload.StartTime)
		if err == nil {
			h.repo.UpdateConsultationSchedule(ctx, bookingUID, startTime)
		}

	case "BOOKING_NO_SHOW_UPDATED":
		// NOTA: ainda não testamos um payload real desse evento (webhook só
		// será registrado de fato após o deploy) — a lógica abaixo assume
		// que o array "attendees" identifica quem faltou; se o cliente faltou,
		// conta como remarcação (política de 2 grátis); se foi o Guerreiro
		// (ausência do organizador), não penaliza o cliente.
		clientNoShow := false
		for _, a := range event.Payload.Attendees {
			if a.NoShow {
				clientNoShow = true
			}
		}
		if clientNoShow {
			h.repo.MarkNoShowAndIncrement(ctx, bookingUID, "no_show_cliente")
		} else {
			h.repo.UpdateConsultationStatus(ctx, bookingUID, "no_show_consultor")
		}

	default:
		log.Printf("evento Cal.com ignorado: %s", event.TriggerEvent)
	}

	w.WriteHeader(http.StatusOK)
}