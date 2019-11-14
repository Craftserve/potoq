package packets

type ConnState VarInt

const (
	HANDSHAKING ConnState = 0
	STATUS      ConnState = 1
	LOGIN       ConnState = 2
	PLAY        ConnState = 3
)

func (state ConnState) String() string {
	switch state {
	case HANDSHAKING:
		return "HANDSHAKING"
	case STATUS:
		return "STATUS"
	case LOGIN:
		return "LOGIN"
	case PLAY:
		return "PLAY"
	default:
		return "UNKNOWN"
	}
}
