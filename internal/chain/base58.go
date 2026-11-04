package chain

import (
	"crypto/sha256"
	"math/big"
	"regexp"
	"strings"
)

const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

var ethAddress = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

func decodeBase58(s string) ([]byte, bool) {
	n := new(big.Int)
	for _, r := range s {
		i := strings.IndexRune(alphabet, r)
		if i < 0 {
			return nil, false
		}
		n.Mul(n, big.NewInt(58))
		n.Add(n, big.NewInt(int64(i)))
	}
	out := n.Bytes()
	zeros := len(s) - len(strings.TrimLeft(s, "1"))
	return append(make([]byte, zeros), out...), true
}

func encodeBase58(b []byte) string {
	n := new(big.Int).SetBytes(b)
	var out []byte
	for mod := new(big.Int); n.Sign() > 0; {
		n.DivMod(n, big.NewInt(58), mod)
		out = append(out, alphabet[mod.Int64()])
	}
	for _, x := range b {
		if x != 0 {
			break
		}
		out = append(out, alphabet[0])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}

// tronHexToBase58 переводит адрес Tron из hex (41 и 20 байт) в привычный вид с T; ошибка проверяется по контрольной сумме.
func tronHexToBase58(raw []byte) string {
	sum := sha256.Sum256(raw)
	sum = sha256.Sum256(sum[:])
	return encodeBase58(append(append([]byte{}, raw...), sum[:4]...))
}

// Valid проверяет формат адреса сети: контрольную сумму Tron, длину Solana, hex Ethereum.
func Valid(ch Chain, address string) bool {
	switch ch {
	case Ethereum:
		return ethAddress.MatchString(address)
	case Tron:
		b, ok := decodeBase58(address)
		if !ok || len(b) != 25 || b[0] != 0x41 {
			return false
		}
		return tronHexToBase58(b[:21]) == address
	case Solana:
		b, ok := decodeBase58(address)
		return ok && len(b) == 32
	}
	return false
}
