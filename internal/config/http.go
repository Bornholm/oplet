package config

import "time"

type HTTP struct {
	BaseURL string  `env:"BASE_URL,expand" envDefault:"/"`
	Address string  `env:"ADDRESS,expand" envDefault:":3002"`
	Authn   Authn   `envPrefix:"AUTHN_"`
	Session Session `envPrefix:"SESSION_"`
}
type Authn struct {
	Providers         AuthProviders `envPrefix:"PROVIDERS_"`
	DefaultAdminEmail string        `env:"DEFAULT_ADMIN_EMAIL,expand"`
	InactiveByDefault bool          `env:"INACTIVE_BY_DEFAULT,expand" envDefault:"true"`
}

type Session struct {
	Keys   []string `env:"KEYS,expand" envSeparator:","`
	Cookie Cookie   `envPrefix:"COOKIE_"`
}

type Cookie struct {
	Path     string        `env:"PATH,expand" envDefault:"/"`
	HTTPOnly bool          `env:"HTTP_ONLY,expand" envDefault:"true"`
	Secure   bool          `env:"SECURE,expand" envDefault:"false"`
	MaxAge   time.Duration `env:"MAX_AGE,expand" enDefault:"24h"`
}

type AuthProviders struct {
	Google OAuth2Provider `envPrefix:"GOOGLE_"`
	Github OAuth2Provider `envPrefix:"GITHUB_"`
	Gitea  GiteaProvider  `envPrefix:"GITEA_"`
	OIDC   OIDCProvider   `envPrefix:"OIDC_"`
}

type OAuth2Provider struct {
	Key    string   `env:"KEY,expand"`
	Secret string   `env:"SECRET,expand"`
	Scopes []string `env:"SCOPES,expand" envSeparator:"," envDefault:"openid,profile,email"`
}

type OIDCProvider struct {
	OAuth2Provider
	DiscoveryURL string `env:"DISCOVERY_URL,expand"`
	Icon         string `env:"ICON,expand"`
	Label        string `env:"LABEL,expand"`
}

type GiteaProvider struct {
	OAuth2Provider
	TokenURL   string `env:"TOKEN_URL,expand"`
	AuthURL    string `env:"AUTH_URL,expand"`
	ProfileURL string `env:"PROFILE_URL,expand"`
	Label      string `env:"LABEL,expand"`
}
