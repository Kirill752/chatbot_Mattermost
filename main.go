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

	// err = strg.SavePool(&pool.Pool{ID: 1, Title: "Cats or Dogs",
	// 	Variants: []string{"Cats", "Dogs"}, Finished: false})
	// if err != nil {
	// 	log.Println(err)
	// }
	// err = strg.AddVote(1, "utnjpkssjpyrxy3oeu7nsotw1e", "Cats")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = strg.FinishPool(1)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// err = strg.DeleteAllVotes(1)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// err = strg.DeletePool(1)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	strg.SelectSomePools(0, 100)
	strg.SelectSomeVotes(0, 100)
	// tables := strg.GetAllTables()
	// for _, v := range tables {
	// 	fmt.Println(v)
	// }
	// TODO: Инициализация обработчика команд
	// TODO: Запуск прослушивания событий
}
