package giris2

type ItfTokenStore interface {
	SetToken(key string, t *Token)
	GetToken(key string) (t *Token, has bool)
	DelToken(key string)
	ClearExpiredToken()
}
