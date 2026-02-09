package main

func main() {
	if err := newRootCmd().Execute(); err != nil {
		exitf("%v", err)
	}
}
