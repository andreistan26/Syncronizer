package file_level

func CheckErr(err error) {
	if err != nil {
		panic(err)
	}
}
