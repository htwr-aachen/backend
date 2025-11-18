package config

type Config struct {
	Global        Global        `koanf:"global"`
	Database      DB            `koanf:"database"`
	Session       Session       `koanf:"session"`
	Public        Public        `koanf:"public"`
	QA            QA            `koanf:"qa"`
	Admin         Admin         `koanf:"admin"`
	Panikzettel   Panikzettel   `koanf:"panikzettel"`
	MetricsServer MetricsServer `koanf:"metrics"`
}
