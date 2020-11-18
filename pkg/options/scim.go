package options

type (
	SCIMOpt struct {
		Enabled bool   `env:"SCIM_ENABLED"`
		BaseURL string `env:"SCIM_BASE_URL"`
		Secret  string `env:"SCIM_SECRET"`
	}
)

func SCIM() (o *SCIMOpt) {
	o = &SCIMOpt{
		Enabled: false,
		BaseURL: "/scim",
	}

	fill(o)

	return
}
