package server

import "hilldan/mqtt/packet"

// if a Client subscribes to “sport/tennis/player1/#”, it would receive messages
// published using these topic names:
//  “sport/tennis/player1”
//  “sport/tennis/player1/ranking”
//  “sport/tennis/player1/score/wimbledon”
//  “sport/#” also matches the singular “sport”, since # includes the parent level.
//  “#” is valid and will receive every Application Message
//  “sport/tennis/#” is valid
//  “sport/tennis#” is not valid
//  “sport/tennis/#/ranking” is not valid
//
// For example, “sport/tennis/+” matches “sport/tennis/player1” and “sport/tennis/player2”,
// but not “sport/tennis/player1/ranking”.
// Also, because the single-level wildcard matches only a single level,
//  “sport/+” does not match “sport” but it does match “sport/”.
//  “+” is valid
//  “+/tennis/#” is valid
//  “sport+” is not valid
//  “sport/+/player1” is valid
//  “/finance” matches “+/+” and “/+”, but not “+”
//
// A subscription to “#” will not receive any messages published to a topic beginning with a
// $
//  A subscription to “+/monitor/Clients” will not receive any messages published to
// “$SYS/monitor/Clients”
//  A subscription to “$SYS/#” will receive messages published to topics beginning with
// “$SYS/”
//  A subscription to “$SYS/monitor/+” will receive messages published to
// “$SYS/monitor/Clients”
//  For a Client to receive messages from topics that begin with $SYS/ and from topics that
// don’t begin with a $, it has to subscribe to both “#” and “$SYS/#”
func check(sub string) (valid bool, path []string) {
	if len(sub) == 0 {
		return
	}
	//split
	j := 0
	for i := 0; i < len(sub); i++ {
		switch sub[i] {
		case '+', '#', '/':
			if i > j {
				path = append(path, sub[j:i])
			}
			path = append(path, string(sub[i]))
			j = i + 1
		case 0:
			return
		}
	}
	if j < len(sub) {
		path = append(path, sub[j:])
	}

	//check
	if len(path) == 1 {
		if path[0] == "/" {
			return
		}
		valid = true
		return
	}
	for i := 1; i < len(path); i++ {
		switch path[i] {
		case "/":
			switch path[i-1] {
			case "/", "#":
				return
			}
		default:
			if path[i-1] != "/" {
				return
			}
		}
	}
	valid = true
	return
}

func matchPath(path1, path2 []string) bool {
	l, ll := len(path1), len(path2)
	if l == 0 || ll == 0 {
		return false
	}
	if path1[0] == "/" && path2[0] != "/" {
		return false
	}
	if (path1[0][0] == '$' || path2[0][0] == '$') && path1[0] != path2[0] {
		return false
	}

	switch l {
	case 1:
		// #, +, xxx
		switch path1[0] {
		case "#":
			return true
		case "+":
			return ll == 1
		default:
			return ll == 1 && path2[0] == path1[0]
		}
	case 2:
		if path1[0] == "/" {
			// /#, /+, /xxx
			switch path1[1] {
			case "#":
				return true
			case "+":
				return ll == 2 && path2[0] == "/"
			default:
				return ll == 2 && path2[1] == path1[1] && path2[0] == "/"
			}
		} else {
			// +/, xxx/
			switch path1[0] {
			case "+":
				return ll == 2 && path2[1] == "/"
			default:
				return ll == 2 && path2[0] == path1[0]
			}
		}
	case 3:
		if path1[0] == "/" {
			// /+/, /xxx/
			switch path1[1] {
			case "+":
				return ll == 3 && path2[0] == "/"
			default:
				return ll == 3 && path2[0] == "/" && path2[1] == path1[1]
			}
		} else {
			// +/+, +/xxx, +/#, xxx/+, xxx/#, xxx/yyy
			switch path1[0] {
			case "+":
				switch path1[2] {
				case "+":
					return ll > 1 && path2[ll-2] == "/"
				case "#":
					return true
				default:
					return ll > 1 && path2[ll-2] == "/" && path2[ll-1] == path1[2]
				}
			default:
				switch path1[2] {
				case "+":
					return ll < 4 && ll > 1 && path2[0] == path1[0]
				case "#":
					return ll > 0 && path2[0] == path1[0]
				default:
					return ll == 3 && path2[0] == path1[0] && path2[2] == path1[2]
				}
			}
		}
	}

	if ll < 2 {
		return false
	}

	// align '/'
	f := func(p1, p2 []string) bool {
		l1, l2 := len(p1), len(p2)
		for k := 0; k < l1 && k < l2; k++ {
			if p1[k] != p2[k] && p1[k] != "+" && p1[k] != "#" {
				return false
			}
		}
		if l1 > l2 {
			return l1-l2 <= 2 && (p1[l1-1] == "#" || p1[l1-1] == "+")
		}
		if l1 < l2 {
			return p1[l1-1] == "#"
		}
		return true
	}
	i, j := 0, 0
	if path1[1] == "/" {
		i = 1
	}
	if path2[1] == "/" {
		j = 1
	}
	switch i - j {
	case 0:
		return f(path1, path2)
	case 1:
		if path1[0] != "+" {
			return false
		}
		return f(path1[1:], path2)
	default:
		return false
	}
}

func match(subs []packet.TopicFilter, topic string) (matched bool, maxQos packet.Bit2) {
	// fmt.Println("match():", subs, topic)
	for _, v := range subs {
		if matchOne(string(v.Topic), topic) {
			matched = true
			maxQos = v.Qos
			return
		}
	}
	return
}

func matchOne(sub, topic string) bool {
	p1, err := WildcardRegistry.Get(sub)
	if err != nil {
		return false
	}
	p2, err := WildcardRegistry.Get(topic)
	if err != nil {
		return false
	}

	return matchPath(p1, p2)
}

// compare compares 2 subscription topic
func compare(path, path2 []string) (flag int, relate bool) {
	if len(path) == len(path2) {
		relate = true
		for k, v := range path {
			if v != path2[k] {
				relate = false
				break
			}
		}
	}
	if relate {
		return
	}

	if matchPath(path, path2) {
		flag = 1
		relate = true
		return
	}
	if matchPath(path2, path) {
		flag = -1
		relate = true
		return
	}
	return
}
