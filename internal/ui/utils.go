package ui

func isInBounds(x int, y int, b bounds) bool {
	return x > b.x1 && x < b.x2 && y > b.y1 && y < b.y2
}
