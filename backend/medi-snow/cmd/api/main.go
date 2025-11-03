package main

// TODO Load from env/config
const PORT = "8080"

func main() {
	// Create app
	app := NewApp()

	// Start server
	if err := app.Run(":" + PORT); err != nil {
		panic(err)
	}
}
