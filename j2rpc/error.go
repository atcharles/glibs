package j2rpc

// declared ...
const (
	ErrParse          ErrorCode = -32700
	ErrInvalidRequest ErrorCode = -32600
	ErrNoMethod       ErrorCode = -32601
	ErrBadParams      ErrorCode = -32602
	ErrInternal       ErrorCode = -32603
	ErrServer         ErrorCode = -32000

	ErrAuthorization ErrorCode = 401
	ErrForbidden     ErrorCode = 403
)

// Error ... Error codes
type Error struct {
	Code    ErrorCode   `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Error) Error() string { return e.Message }

func (e *Error) ErrorCode() int { return int(e.Code) }

func (e *Error) ErrorData() interface{} { return e.Data }

// ErrorCode ... Error codes
type ErrorCode int

type IRPCError interface {
	RPCError() *Error
}

type ItfJ2rpcError interface {
	ErrorCode() int
	Error() string
	ErrorData() interface{}
}

type (
	//TokenError ...
	TokenError string
	//ForbiddenError ...
	ForbiddenError string
)

func (t TokenError) Error() string { return string(t) }

func (e ForbiddenError) Error() string { return string(e) }

// NewError ...
func NewError(code ErrorCode, Msg string, data ...interface{}) *Error {
	ee := &Error{Code: code, Message: Msg}
	if len(data) > 0 {
		ee.Data = data[0]
	}
	return ee
}