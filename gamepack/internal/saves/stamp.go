package saves

import "time"

func stamp() string { return time.Now().Format("20060102-150405") }
