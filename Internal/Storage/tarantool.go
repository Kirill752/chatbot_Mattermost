package storage

import (
	"fmt"
	"vk_tarantool/Internal/handlers/pool"

	"github.com/tarantool/go-tarantool"
)

type Storage struct {
	db *tarantool.Connection
}

func New(address string, user string, password string) (*Storage, error) {
	const op = "storage.tarantool.New"
	opts := tarantool.Opts{User: user, Pass: password}
	conn, err := tarantool.Connect(address, opts)
	if err != nil {
		return nil, fmt.Errorf("%s: unable to connect %w", op, err)
	}
	_, err = conn.Call("box.schema.space.create", []interface{}{
		"pools",
		map[string]bool{"if_not_exists": true},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create table %w", op, err)
	}
	_, err = conn.Call("box.space.pools:format", [][]map[string]string{
		{
			{"name": "id", "type": "number"},
			{"name": "title", "type": "string"},
			{"name": "creator_id", "type": "number"},
			{"name": "variants", "type": "map"},
		}})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create coloums %w", op, err)
	}
	_, err = conn.Call("box.space.pools:create_index", []interface{}{
		"primary",
		map[string]interface{}{
			"parts":         []string{"id"},
			"if_not_exists": true}})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create index %w", op, err)
	}
	return &Storage{db: conn}, nil
}

func (s *Storage) CloseConnection() {
	s.db.Close()
}

func (s *Storage) Save(p *pool.Pool) error {
	const op = "storage.tarantool.Save"
	variants := make(map[string]int, len(p.Variants))
	for _, v := range p.Variants {
		variants[v] = 0
	}
	_, err := s.db.Insert("pools", []any{p.ID, p.Title, 1, variants})
	if err != nil {
		return fmt.Errorf("%s: unable to save pool %w", op, err)
	}
	return nil
}

func (s *Storage) Select(id int) (*tarantool.Response, error) {
	const op = "storage.tarantool.Select"
	data, err := s.db.Do(
		tarantool.NewSelectRequest("pools").
			Iterator(tarantool.IterEq).
			Key([]any{uint(id)}),
	).Get()
	if err != nil {
		return nil, fmt.Errorf("%s: unable to select pool whit id %d: %w", op, id, err)
	}
	return data, nil
}

func (s *Storage) Delete(id int) error {
	const op = "storage.tarantool.Delete"
	_, err := s.db.Do(
		tarantool.NewDeleteRequest("pools").
			Key([]any{uint(id)}),
	).Get()
	if err != nil {
		return fmt.Errorf("%s: unable to delete pool whit id %d: %w", op, id, err)
	}
	return nil
}

func (s *Storage) SelectSome(offset uint32, limit uint32) error {
	const op = "storage.tarantool.SelectSome"
	resp, err := s.db.Select("pools", "primary", offset, limit, tarantool.IterAll, []any{})
	if err != nil {
		return fmt.Errorf("%s: unable to select %w", op, err)
	}
	fmt.Println("Все опросы:")
	fmt.Println("| ID | Вопрос | Создатель | Варианты |")
	for _, item := range resp.Data {
		data := item.([]any)
		fmt.Printf("|%v |%v |%v |%v |\n", data[0], data[1], data[2], data[3])
	}
	return nil
}

func (s *Storage) AddVote(id int, variant string) error {
	const op = "storage.tarantool.AddVote"
	resp, err := s.Select(id)
	if err != nil {
		return fmt.Errorf("%s: selection failed %w", op, err)
	}
	if len(resp.Data) == 0 {
		return fmt.Errorf("%s: pool not found %w", op, err)
	}
	poll := resp.Data[0].([]any)
	variants, ok := poll[3].(map[any]any)
	if !ok {
		return fmt.Errorf("%s: error type assertion %w", op, err)
	}
	if val, ok := variants[variant].(uint64); ok {
		variants[variant] = val + 1
	} else {
		return fmt.Errorf("%s: value is not int %w", op, err)
	}
	fmt.Println(variants)
	_, err = s.db.Replace("pools", poll)
	if err != nil {
		return fmt.Errorf("%s: replace failed %w", op, err)
	}
	return nil
}

func (s *Storage) CancelVote(id int, variant string) error {
	const op = "storage.tarantool.CancelVote"
	resp, err := s.Select(id)
	if err != nil {
		return fmt.Errorf("%s: selection failed %w", op, err)
	}
	if len(resp.Data) == 0 {
		return fmt.Errorf("%s: pool not found %w", op, err)
	}
	poll := resp.Data[0].([]any)
	variants, ok := poll[3].(map[any]any)
	if !ok {
		return fmt.Errorf("%s: error type assertion %w", op, err)
	}
	if val, ok := variants[variant].(uint64); ok {
		variants[variant] = val - 1
	} else {
		return fmt.Errorf("%s: value is not int %w", op, err)
	}
	fmt.Println(variants)
	_, err = s.db.Replace("pools", poll)
	if err != nil {
		return fmt.Errorf("%s: replace failed %w", op, err)
	}
	return nil
}
