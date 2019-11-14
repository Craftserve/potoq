package packets

type Direction uint8

const (
	ServerBound Direction = 1
	ClientBound Direction = 2
)

func (direction Direction) String() string {
	switch direction {
	case ServerBound:
		return "SB"
	case ClientBound:
		return "CB"
	default:
		panic("invalid direction")
	}
}
