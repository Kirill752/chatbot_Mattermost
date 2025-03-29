package storage

import (
	"fmt"
	"log"
	"vk_tarantool/Internal/handlers/pool"
	"vk_tarantool/Internal/handlers/vote"

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
	/*Создание таблицы с опросами*/
	_, err = conn.Call("box.schema.space.create", []any{
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
			{"name": "creator", "type": "string"},
			{"name": "variants", "type": "map"},
			{"name": "finished", "type": "boolean"},
		}})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create coloums %w", op, err)
	}
	_, err = conn.Call("box.space.pools:create_index", []any{
		"primary",
		map[string]any{
			"parts":         []string{"id"},
			"if_not_exists": true}})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create index %w", op, err)
	}

	/*Создание таблицы с голосами*/
	_, err = conn.Call("box.schema.space.create", []any{
		"votes",
		map[string]bool{"if_not_exists": true},
	})
	if err != nil {
		return nil, fmt.Errorf(`%s: unable to create "votes" table %w`, op, err)
	}
	_, err = conn.Call("box.space.votes:format", [][]map[string]string{
		{
			{"name": "pool_id", "type": "number"},
			{"name": "user", "type": "string"},
			{"name": "variant", "type": "string"},
		}})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create coloums %w", op, err)
	}
	_, err = conn.Call("box.space.votes:create_index", []any{
		"primary",
		map[string]any{
			"parts":         []string{"pool_id", "user"},
			"if_not_exists": true}})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create index %w", op, err)
	}

	// Вторичный индекс
	_, err = conn.Call("box.space.votes:create_index", []any{
		"pool_id",
		map[string]any{
			"parts":         []string{"pool_id"},
			"if_not_exists": true,
			"type":          "TREE",
			"unique":        false,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("%s: unable to create index: %w", op, err)
	}

	return &Storage{db: conn}, nil
}

func (s *Storage) CloseConnection() {
	s.db.Close()
}

func (s *Storage) DeleteTable(tableName string) {
	// Выполнение Lua-кода для удаления таблицы
	resp, err := s.db.Eval("box.space."+tableName+":drop()", []any{})
	if err != nil {
		log.Fatalf("Error dropping table: %s", err)
	}

	log.Printf("Table %s dropped successfully. Response: %+v", tableName, resp)
}

///////////////////////////////////////////////////////////////////////////////

func (s *Storage) SavePool(p *pool.Pool) error {
	const op = "storage.tarantool.SavePool"
	variants := make(map[string]int, len(p.Variants))
	for _, v := range p.Variants {
		variants[v] = 0
	}
	_, err := s.db.Insert("pools", []any{p.ID, p.Title, p.Creator, variants, p.Finished})
	if err != nil {
		return fmt.Errorf("%s: unable to save pool %w", op, err)
	}
	return nil
}

func (s *Storage) SelectPool(id uint) (*tarantool.Response, error) {
	const op = "storage.tarantool.SelectPool"
	data, err := s.db.Do(
		tarantool.NewSelectRequest("pools").
			Iterator(tarantool.IterEq).
			Key([]any{id}),
	).Get()
	if err != nil {
		return nil, fmt.Errorf("%s: unable to select pool whit id %d: %w", op, id, err)
	}
	return data, nil
}

func (s *Storage) DeletePool(id uint) error {
	const op = "storage.tarantool.DeletePool"
	_, err := s.db.Do(
		tarantool.NewDeleteRequest("pools").
			Key([]any{uint(id)}),
	).Get()
	if err != nil {
		return fmt.Errorf("%s: unable to delete pool whit id %d: %w", op, id, err)
	}
	return nil
}

func (s *Storage) FinishPool(poolId uint, user string) error {
	const op = "storage.tarantool.FinishPool"

	resp, err := s.SelectPool(poolId)
	if err != nil || len(resp.Data) == 0 {
		return fmt.Errorf("%s: pool not found", op)
	}
	updated := resp.Data[0].([]any)
	if creator, ok := updated[2].(string); ok {
		if creator != user {
			return fmt.Errorf("not creator")
		}
	} else {
		return fmt.Errorf("%s: %w", op, err)
	}
	// TODO: может быть сделать просто смену состояния на противоположное
	updated[4] = true

	_, err = s.db.Replace("pools", updated)
	return err
}

func (s *Storage) SelectSomePools(offset uint32, limit uint32) error {
	const op = "storage.tarantool.SelectSomePools"
	resp, err := s.db.Select("pools", "primary", offset, limit, tarantool.IterAll, []any{})
	if err != nil {
		return fmt.Errorf("%s: unable to select %w", op, err)
	}
	fmt.Println("POOLS:")
	fmt.Println("| ID | QESTION | CREATOR_ID | VARIANTS | FINISHED |")
	for _, item := range resp.Data {
		data := item.([]any)
		fmt.Printf("|%v |%v |%v |%v |%v |\n", data[0], data[1], data[2], data[3], data[4])
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////

func (s *Storage) SaveVote(v *vote.Vote) error {
	const op = "storage.tarantool.SaveVote"
	_, err := s.db.Replace("votes", []any{v.PoolID, v.UserId, v.Variant})
	if err != nil {
		return fmt.Errorf("%s: unable to save pool %w", op, err)
	}
	return nil
}

func (s *Storage) SelectVote(poolId uint, user string) (*tarantool.Response, error) {
	const op = "storage.tarantool.SelectVote"
	data, err := s.db.Do(
		tarantool.NewSelectRequest("votes").
			Iterator(tarantool.IterEq).
			Key([]any{poolId, user}),
	).Get()
	if err != nil {
		return nil, fmt.Errorf("%s: unable to select vote: %w", op, err)
	}
	return data, nil
}

func (s *Storage) DeleteVote(poolId uint, user string) error {
	const op = "storage.tarantool.DeleteVote"
	_, err := s.db.Do(
		tarantool.NewDeleteRequest("votes").
			Key([]any{poolId, user}),
	).Get()
	if err != nil {
		return fmt.Errorf("%s: unable to delete vote: %w", op, err)
	}
	return nil
}

func (s *Storage) DeleteAllVotes(poolId uint) error {
	const op = "storage.tarantool.DeleteAllVotesInPool"

	script := `
    local pool_id = ...
    local space = box.space.votes
    local count = 0

    -- Используем вторичный индекс для поиска
    for _, tuple in space.index.pool_id:pairs(pool_id) do
        space:delete{tuple[1], tuple[2]}  -- Удаляем по первичному ключу
        count = count + 1
    end

    return count
    `
	_, err := s.db.Eval(script, []any{float64(poolId)})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// func (s *Storage) GetIndexes(spaceName string) (map[string]any, error) {
// 	const op = "storage.tarantool.GetIndexes"

// 	script := `
//     local space_name = ...
//     local space = box.space[space_name]
//     if not space then
//         return nil, "Space not found"
//     end

//     local indexes = {}
//     for name, index in pairs(space.index) do
//         indexes[name] = {
//             id = index.id,
//             type = index.type,
//             unique = index.unique,
//             parts = index.parts
//         }
//     end

//     return indexes
//     `

// 	resp, err := s.db.Eval(script, []any{spaceName})
// 	if err != nil {
// 		return nil, fmt.Errorf("%s: %w", op, err)
// 	}

// 	if result, ok := resp.Data[0].(map[any]any); ok {
// 		converted := make(map[string]any)
// 		for k, v := range result {
// 			if key, ok := k.(string); ok {
// 				converted[key] = v
// 			}
// 		}
// 		return converted, nil
// 	}

// 	return nil, fmt.Errorf("%s: unexpected response format", op)
// }

func (s *Storage) SelectSomeVotes(offset uint32, limit uint32) error {
	const op = "storage.tarantool.SelectSomeVotes"
	resp, err := s.db.Select("votes", "primary", offset, limit, tarantool.IterAll, []any{})
	if err != nil {
		return fmt.Errorf("%s: unable to select %w", op, err)
	}
	fmt.Println("VOTES:")
	fmt.Println("| POOL_ID | USER | VARIANT |")
	for _, item := range resp.Data {
		data := item.([]any)
		fmt.Printf("|%v |%v |%v |\n", data[0], data[1], data[2])
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////

func (s *Storage) AddVote(id uint, user string, variant string) error {
	const op = "storage.tarantool.AddVote"
	// Идем в таблицу "Pools"
	resp, err := s.SelectPool(id)
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

	// Идем в таблицу "Votes"
	// Смотрим, голосовал ли уже пользователь  в этом опросе
	resp, err = s.SelectVote(id, user)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	// Если записи в таблице "votes" еще нет, то
	if len(resp.Data) != 0 {
		// Уменьшаем стрый вариант в табличке "Pools" на 1
		// Старый вариант
		vote := resp.Data[0].([]any)
		oldVariant := vote[2].(string)
		if val, ok := variants[oldVariant].(uint64); ok {
			if val <= 0 {
				return fmt.Errorf("%s: incorrect value <= 0 %w", op, err)
			}
			variants[oldVariant] = val - 1
		} else {
			return fmt.Errorf("%s: value is not uint64 %w", op, err)
		}
	}
	// Создаем запись в таблице votes
	err = s.SaveVote(&vote.Vote{PoolID: id, UserId: user, Variant: variant})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	// Прибавляем 1 к варианту в таблице "Pools"
	if val, ok := variants[variant].(uint64); ok {
		variants[variant] = val + 1
	} else {
		return fmt.Errorf("%s: value is not uint64 %w", op, err)
	}
	// Замена результата
	poll[3] = variants
	_, err = s.db.Replace("pools", poll)
	if err != nil {
		return fmt.Errorf("%s: replace failed %w", op, err)
	}
	return nil
}

func (s *Storage) CancelVote(id uint, variant string) error {
	const op = "storage.tarantool.CancelVote"
	resp, err := s.SelectPool(id)
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

func (s *Storage) GetAllTables() []any {
	// Получение списка всех таблиц
	resp, err := s.db.Eval(`
		local result = {}
		for _, space in box.space._space:pairs() do
			table.insert(result, space.name)
		end
		return result
	`, []any{})

	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	return resp.Data[0].([]any)
}
