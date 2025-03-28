package main

import (
	"log"
	storage "vk_tarantool/Internal/Storage"
)

func main() {
	// TODO: Загрузка конфига
	// TODO: Инициализация mattermost
	// app := application.New("./Config/conf.yaml")
	// app.SetupGracefulShutdown()
	// app.ListenToEvents()
	// TODO: Инициализация tarantool
	strg, err := storage.New("127.0.0.1:3301", "storage", "passw0rd")
	if err != nil {
		log.Fatal(err)
	}
	defer strg.CloseConnection()
	// opts := tarantool.Opts{User: "storage", Pass: "passw0rd"}
	// conn, err := tarantool.Connect("127.0.0.1:3301", opts)
	// if err != nil {
	// 	panic(err)
	// }
	// defer conn.Close()
	// resp, err := conn.Call("box.space._space:select", []any{tarantool.IterAll})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// for _, tuple := range resp.Data {
	// 	space := tuple.([]any)
	// 	fmt.Printf("ID: %d, Name: %s\n", space[0].(uint32), space[2].(string))
	// }
	// msg := `pool 1 "Что делаете?" Сплю Ем`
	// pl, err := pool.Create(msg)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = strg.Save(pl)
	// if err != nil {
	// 	panic(err)
	// }
	// err = strg.CancelVote(1, "Ем")
	// if err != nil {
	// 	panic(err)
	// }
	// err = strg.Delete(3)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// d, err := strg.Select(2)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(d)
	strg.SelectSome(0, 100)
	// getAllPolls(conn)
	// TODO: Инициализация обработчика команд
	// TODO: Запуск прослушивания событий
	// // msg := `опРос 1 "ваш любимый фрукт?" яблоко груша киви мандарин апельсин`
	// // pool, err := pool.CreatePool(msg)
	// // if err != nil {
	// // 	log.Fatal(err)
	// // }
	// // fmt.Println(pool)
}
