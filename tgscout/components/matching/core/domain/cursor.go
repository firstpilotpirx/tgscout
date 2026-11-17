package domain

// Cursor is the position of one chat in memory: the highest message ID below
// which every begun message is finished. It never moves back and never
// passes a message that is begun and not finished.
type Cursor struct {
	position int
	inFlight map[int]bool
	finished map[int]bool
}

// NewCursor returns a cursor at no position.
func NewCursor() *Cursor {
	return &Cursor{inFlight: map[int]bool{}, finished: map[int]bool{}}
}

// Begin records a message that is being judged.
func (c *Cursor) Begin(id int) { c.inFlight[id] = true }

// Finish records a finished message and returns the position and whether it moved.
func (c *Cursor) Finish(id int) (int, bool) {
	delete(c.inFlight, id)
	c.finished[id] = true
	lowestInFlight, blocked := c.lowestInFlight()
	next := c.position
	for finished := range c.finished {
		if (!blocked || finished < lowestInFlight) && finished > next {
			next = finished
		}
	}
	for finished := range c.finished {
		if finished <= next {
			delete(c.finished, finished)
		}
	}
	moved := next > c.position
	c.position = next
	return c.position, moved
}

func (c *Cursor) lowestInFlight() (int, bool) {
	lowest, found := 0, false
	for id := range c.inFlight {
		if !found || id < lowest {
			lowest, found = id, true
		}
	}
	return lowest, found
}
