package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type DatabaseConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Name            string        `yaml:"name"`
	SSLMode         string        `yaml:"sslmode"`
	MaxConns        int           `yaml:"max_conns"`
	MinConns        int           `yaml:"min_conns"`
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"`
}

// DSN includes the pool settings; pgxpool reads them from the URL.
func (d DatabaseConfig) DSN() string {
	params := url.Values{}
	params.Set("sslmode", d.SSLMode)
	if d.MaxConns > 0 {
		params.Set("pool_max_conns", strconv.Itoa(d.MaxConns))
	}
	if d.MinConns > 0 {
		params.Set("pool_min_conns", strconv.Itoa(d.MinConns))
	}
	if d.MaxConnLifetime > 0 {
		params.Set("pool_max_conn_lifetime", d.MaxConnLifetime.String())
	}
	if d.MaxConnIdleTime > 0 {
		params.Set("pool_max_conn_idle_time", d.MaxConnIdleTime.String())
	}

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.User, d.Password),
		Host:     net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		Path:     "/" + d.Name,
		RawQuery: params.Encode(),
	}
	return dsn.String()
}

type ReservationConfig struct {
	TTLSeconds int `yaml:"ttl_seconds"`
}

type WorkerConfig struct {
	SweeperInterval  time.Duration `yaml:"sweeper_interval"`
	SweeperBatchSize int           `yaml:"sweeper_batch_size"`
}

type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Database    DatabaseConfig    `yaml:"database"`
	Reservation ReservationConfig `yaml:"reservation"`
	Worker      WorkerConfig      `yaml:"worker"`
}

// Environment variables override the file.
func Load(path string) (*Config, error) {
	if path == "" {
		path = "config.yaml"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read configuration file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	envString(&cfg.Server.Host, "SERVER_HOST")
	envInt(&cfg.Server.Port, "SERVER_PORT")
	envString(&cfg.Database.Host, "DB_HOST")
	envInt(&cfg.Database.Port, "DB_PORT")
	envString(&cfg.Database.User, "DB_USER")
	envString(&cfg.Database.Password, "DB_PASSWORD")
	envString(&cfg.Database.Name, "DB_NAME")
	envString(&cfg.Database.SSLMode, "DB_SSLMODE")

	if cfg.Database.SSLMode == "" {
		cfg.Database.SSLMode = "disable"
	}
	return &cfg, nil
}

func envString(dst *string, key string) {
	if v := os.Getenv(key); v != "" {
		*dst = v
	}
}

func envInt(dst *int, key string) {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		*dst = v
	}
}
