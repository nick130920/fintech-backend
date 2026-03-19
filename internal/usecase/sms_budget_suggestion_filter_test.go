package usecase

import "testing"

func TestLikelyBankTransactionSMS_ExcludeNoise(t *testing.T) {
	excluded := []string{
		"Bancolombia: Alguno de los datos que ingresaste de tu tarjeta 9401 en Google Pay esta incorrecto.",
		"Aviso Movistar: Por mora de $30000 en tu equipo Movistar 6043098872 se inicio proceso juridico.",
		"Movistar te informa: Tienes 1 productos en mora por 70437.0 Consulta en https://mq.telefonicawebsites.co/x",
		"En #TuExito ahorra con las Superofertas Hasta 30% dcto en ref.",
		"AVANZO: AMPARO ,por ser parte de Recaudo Bogota S.A.S tienes acceso a un beneficio Cupo digital + bono $50K",
		"Estar al dia tu mejor referencia crediticia! Tu factura Movistar Ref.6082103530 por $70.437 con 4 dias mora.",
		"Gana 3.000 CMR Puntos ($30mil) en HOT SALE! Compra +$350mil en Falabella",
		"Telegram code: 24807\n\nYou can also tap on this link",
		"Bancolombia: 0% interes* enMacCenterpor compras de 800.000 o mas pagando a 4, 6, 12 cuotas",
		"Bancolombia: Nicolas, $18.387.529 sumarian tus gastos en 2026. Conocelos en App Mi Bancolombia>Explorar>Dia a Dia>Gastos",
		"Hola Nicolas queremos recordarte que tu pago con Banco Falabella aun esta pendiente.",
		"Nicolas, notamos que tu pago de $ 61.879 con Banco Falabella vencio el 05/03/2026. Evita cargos adicionales",
		"Consulta y paga tu factura aqui: https://f.fcert.co/?a=x",
		"Movistar no envia mensajes con porcentaje de descuento para pagar tu servicio",
		"Rappi: Caja de sobres PANINI en preventa por Rappi.",
		"TUYA: Hola Nicolas No esperes mas para estrenar TELEVISOR.",
		"SURA: ¿Que tiene que ver tu estado de animo con lo que comes? TODO, conoce el porque en Longevo",
		"Gracias por tu compromiso con la educacion. ICETEX sigue Impulsando Oportunidades",
	}
	for _, body := range excluded {
		if likelyBankTransactionSMS(body) {
			t.Errorf("expected excluded (noise), got included: %.120q", body)
		}
	}
}

func TestLikelyBankTransactionSMS_IncludeRealMovements(t *testing.T) {
	included := []string{
		"Bancolombia: Transferiste $23,900.00 por Boton Bancolombia a Wompi SAS desde producto *0058. 18/03/2026 08:11:07",
		"Bancolombia: Transferiste $12,000.00 desde tu cuenta 0058 a la cuenta *3154072167 el 16/03/2026 a las 18:27.",
		"Bancolombia: NICOLAS, recibiste una transferencia de Nicolas Andres Muñoz Coronado por $40,000.00 en tu cuenta *0058",
		"Bancolombia: Compraste COP90.000,00 en LA MANSION DE LAS FL con tu T.Cred *9401, el 16/03/2026 a las 18:17.",
		"Bancolombia: Pagaste $61,880.00 a BANCO FALABELLA S A desde tu producto *0058 el 15/03/2026 19:30:05.",
		"Bancolombia: NICOLAS ANDRES MUÑOZ CORONADO pagaste $10,200.00 por codigo QR desde tu cuenta *0058",
	}
	for _, body := range included {
		if !likelyBankTransactionSMS(body) {
			t.Errorf("expected included (real tx), got excluded: %.120q", body)
		}
	}
}
