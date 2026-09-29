package notify

import "log"

type WhatsAppNotifier struct{}

func NewWhatsAppNotifier() *WhatsAppNotifier {
	return &WhatsAppNotifier{}
}

// SendMessage ainda não integra com a API real da Meta — o número do
// Guerreiro ainda está em migração. Por enquanto só loga, pra o resto do
// fluxo já funcionar; trocamos a implementação aqui dentro quando o
// WhatsApp estiver pronto, sem mexer em mais nada do sistema.
func (w *WhatsAppNotifier) SendMessage(to, message string) error {
	log.Printf("[STUB] enviaria WhatsApp para %s: %s", to, message)
	return nil
}