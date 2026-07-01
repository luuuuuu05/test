package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server ServerConfig
	Game   GameConfig
	Log    LogConfig
}

type ServerConfig struct {
	HTTPPort int
	WSPort   int
	UDPPort  int
}

type GameConfig struct {
	DefaultDuration  int
	DefaultMaxFlags  int
	DefaultMinFlags  int
	RespawnDelay     float64
	GrabDistance     float64
	DoubleDuration   int
	MaxDoubleItems   int
	SceneMapMaxBytes int
	FlagWeights      map[string]int
}

type LogConfig struct {
	Level  string
	Format string
}

func Default() Config {
	return Config{
		Server: ServerConfig{
			HTTPPort: 8080,
			WSPort:   8081,
			UDPPort:  9090,
		},
		Game: GameConfig{
			DefaultDuration:  180,
			DefaultMaxFlags:  5,
			DefaultMinFlags:  4,
			RespawnDelay:     2,
			GrabDistance:     1.5,
			DoubleDuration:   20,
			MaxDoubleItems:   1,
			SceneMapMaxBytes: 256 * 1024,
			FlagWeights: map[string]int{
				"white": 50,
				"red":   35,
				"gold":  15,
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

func Load(path string) (Config, error) {
	cfg := Default()
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	defer f.Close()

	type stackItem struct {
		indent int
		key    string
	}
	var stack []stackItem

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if cut := strings.IndexByte(line, '#'); cut >= 0 {
			line = line[:cut]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		if value == "" {
			stack = append(stack, stackItem{indent: indent, key: key})
			continue
		}
		pathParts := make([]string, 0, len(stack)+1)
		for _, item := range stack {
			pathParts = append(pathParts, item.key)
		}
		pathParts = append(pathParts, key)
		assign(&cfg, strings.Join(pathParts, "."), trimQuotes(value))
	}
	return cfg, scanner.Err()
}

func assign(cfg *Config, key, value string) {
	switch key {
	case "server.http_port":
		cfg.Server.HTTPPort = atoi(value, cfg.Server.HTTPPort)
	case "server.ws_port":
		cfg.Server.WSPort = atoi(value, cfg.Server.WSPort)
	case "server.udp_port":
		cfg.Server.UDPPort = atoi(value, cfg.Server.UDPPort)
	case "game.default_duration":
		cfg.Game.DefaultDuration = atoi(value, cfg.Game.DefaultDuration)
	case "game.default_max_flags":
		cfg.Game.DefaultMaxFlags = atoi(value, cfg.Game.DefaultMaxFlags)
	case "game.default_min_flags":
		cfg.Game.DefaultMinFlags = atoi(value, cfg.Game.DefaultMinFlags)
	case "game.respawn_delay":
		cfg.Game.RespawnDelay = atof(value, cfg.Game.RespawnDelay)
	case "game.grab_distance":
		cfg.Game.GrabDistance = atof(value, cfg.Game.GrabDistance)
	case "game.double_duration":
		cfg.Game.DoubleDuration = atoi(value, cfg.Game.DoubleDuration)
	case "game.max_double_items":
		cfg.Game.MaxDoubleItems = atoi(value, cfg.Game.MaxDoubleItems)
	case "game.scene_map_max_bytes":
		cfg.Game.SceneMapMaxBytes = atoi(value, cfg.Game.SceneMapMaxBytes)
	case "game.flag_weights.white":
		cfg.Game.FlagWeights["white"] = atoi(value, cfg.Game.FlagWeights["white"])
	case "game.flag_weights.red":
		cfg.Game.FlagWeights["red"] = atoi(value, cfg.Game.FlagWeights["red"])
	case "game.flag_weights.gold":
		cfg.Game.FlagWeights["gold"] = atoi(value, cfg.Game.FlagWeights["gold"])
	case "log.level":
		cfg.Log.Level = value
	case "log.format":
		cfg.Log.Format = value
	}
}

func trimQuotes(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, `"`)
	v = strings.Trim(v, `'`)
	return v
}

func atoi(v string, fallback int) int {
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func atof(v string, fallback float64) float64 {
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}
