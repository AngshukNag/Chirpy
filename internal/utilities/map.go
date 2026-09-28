package utilities

func Map[T any, U any] (inputArray []T, f func(input T) U) []U {
	newSlice := make([]U, len(inputArray))

	for index, element := range inputArray {
		newSlice[index] = f(element)
	}

	return newSlice
}