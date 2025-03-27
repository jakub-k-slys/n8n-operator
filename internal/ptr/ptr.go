package ptr

// String returns a pointer to the string value passed in.
func String(s string) *string {
	return &s
}

// Int32 returns a pointer to the int32 value passed in.
func Int32(i int32) *int32 {
	return &i
}
