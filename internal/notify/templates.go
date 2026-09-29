package notify

import "fmt"

func BuildPurchaseEmailHTML(serviceName string) string {
	return fmt.Sprintf(`
		<h1>Pagamento confirmado — %s</h1>
		<p>Seu pagamento foi aprovado com sucesso!</p>
		<p>Em breve o Guerreiro da Fronteira entrará em contato pelo WhatsApp para dar continuidade ao seu atendimento.</p>
	`, serviceName)
}

func BuildPurchaseWhatsAppMessage(serviceName string) string {
	return fmt.Sprintf("Seu pagamento do serviço %s foi confirmado! O Guerreiro da Fronteira vai te chamar por aqui em breve.", serviceName)
}