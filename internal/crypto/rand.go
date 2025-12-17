package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

func GenerateCode(digits uint) (string, error) {
	max := new(big.Int).Exp(
		big.NewInt(10),
		big.NewInt(int64(digits)),
		nil,
	)

	n, err := rand.Int(rand.Reader, max)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0*d", digits, n), nil
}
