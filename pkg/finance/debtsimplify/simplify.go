// Package debtsimplify implementa el algoritmo de simplificación de deudas
// usado para mostrar al usuario el menor número de transferencias entre los
// miembros de un grupo (estilo Splitwise).
package debtsimplify

import (
	"math"
	"sort"
)

// Transfer representa un pago propuesto entre dos miembros para saldar deudas
type Transfer struct {
	From   uint    `json:"from_member_id"`
	To     uint    `json:"to_member_id"`
	Amount float64 `json:"amount"`
}

// Balance representa el saldo neto de un miembro:
//   - Positivo: el miembro PRESTÓ (debe recibir dinero).
//   - Negativo: el miembro DEBE (tiene que pagar).
//   - Cero: está saldado.
type Balance struct {
	MemberID uint
	Net      float64
}

const epsilon = 0.01

// Simplify aplica una estrategia greedy que empareja en cada iteración al
// mayor acreedor con el mayor deudor hasta que todos los saldos converjan a
// cero (con tolerancia de 1 centavo). El número de transferencias generado
// está acotado por N-1, lo que es óptimo en la práctica para el caso de uso.
//
// Los montos se redondean a 2 decimales. Si la suma de balances no es cero
// (por errores de captura), el residuo se distribuye en la última operación.
func Simplify(balances map[uint]float64) []Transfer {
	if len(balances) == 0 {
		return nil
	}

	creditors := make([]Balance, 0, len(balances))
	debtors := make([]Balance, 0, len(balances))

	for memberID, net := range balances {
		rounded := round2(net)
		if rounded > epsilon {
			creditors = append(creditors, Balance{MemberID: memberID, Net: rounded})
		} else if rounded < -epsilon {
			debtors = append(debtors, Balance{MemberID: memberID, Net: rounded})
		}
	}

	// Orden estable para resultados deterministas:
	// acreedores por monto desc y luego id asc; deudores por monto asc (más
	// negativo primero) y luego id asc.
	sort.SliceStable(creditors, func(i, j int) bool {
		if creditors[i].Net != creditors[j].Net {
			return creditors[i].Net > creditors[j].Net
		}
		return creditors[i].MemberID < creditors[j].MemberID
	})
	sort.SliceStable(debtors, func(i, j int) bool {
		if debtors[i].Net != debtors[j].Net {
			return debtors[i].Net < debtors[j].Net
		}
		return debtors[i].MemberID < debtors[j].MemberID
	})

	transfers := make([]Transfer, 0)
	i, j := 0, 0
	for i < len(creditors) && j < len(debtors) {
		credit := &creditors[i]
		debt := &debtors[j]

		amount := math.Min(credit.Net, -debt.Net)
		amount = round2(amount)
		if amount > epsilon {
			transfers = append(transfers, Transfer{
				From:   debt.MemberID,
				To:     credit.MemberID,
				Amount: amount,
			})
			credit.Net = round2(credit.Net - amount)
			debt.Net = round2(debt.Net + amount)
		}

		if credit.Net <= epsilon {
			i++
		}
		if debt.Net >= -epsilon {
			j++
		}
	}

	return transfers
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
