package db

import "fmt"

func NewEngine(engine string) (Engine, error) {
	switch engine {
	case "postgres":
		return &Postgres{}, nil
	case "mysql":
		return &MySQL{}, nil
	default:
		return nil, fmt.Errorf("unsupported engine: %s", engine)
	}
}

type Registry struct {
	engines map[string]Engine
	configs map[string]ConnectionConfig
	active  string
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{
		engines: make(map[string]Engine),
		configs: make(map[string]ConnectionConfig),
	}
}

func (r *Registry) Add(name string, engine Engine, cfg ConnectionConfig) {
	r.engines[name] = engine
	r.configs[name] = cfg
	r.order = append(r.order, name)
	if r.active == "" {
		r.active = name
	}
}

func (r *Registry) Remove(name string) {
	delete(r.engines, name)
	delete(r.configs, name)
	for i, n := range r.order {
		if n == name {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
	if r.active == name {
		if len(r.order) > 0 {
			r.active = r.order[0]
		} else {
			r.active = ""
		}
	}
}

func (r *Registry) Get(name string) Engine {
	return r.engines[name]
}

func (r *Registry) GetConfig(name string) ConnectionConfig {
	return r.configs[name]
}

func (r *Registry) SetActive(name string) {
	if _, ok := r.engines[name]; ok {
		r.active = name
	}
}

func (r *Registry) Active() Engine {
	if r.active == "" {
		return nil
	}
	return r.engines[r.active]
}

func (r *Registry) ActiveName() string {
	return r.active
}

func (r *Registry) Names() []string {
	return r.order
}

func (r *Registry) CloseAll() {
	for _, e := range r.engines {
		e.Close()
	}
}
