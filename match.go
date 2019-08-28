package mqtt

func Match(route, topic string) bool {
	var j int
	for i := 0; i < len(topic); i++ {
		switch route[j] {
		case '#':
			return true
		case '+':
			if topic[i] != '/' {
				continue
			}
			j++
		default:
			if route[j] != topic[i] {
				return false
			}
		}
		re := len(route) <= j+1
		te := len(topic) == i+1
		if !te && re || te && !re {
			return false
		}
		j++
	}
	return true
}
