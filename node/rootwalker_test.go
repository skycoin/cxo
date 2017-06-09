package node

import (
	"testing"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/cipher/encoder"

	"github.com/skycoin/cxo/skyobject"
)

type Board struct {
	Name     string
	Creator  skyobject.Reference `skyobject:"schema=Person"`
	Featured skyobject.Dynamic
	Threads  skyobject.References `skyobject:"schema=Thread"`
}

type Thread struct {
	Name    string
	Creator skyobject.Reference  `skyobject:"schema=Person"`
	Posts   skyobject.References `skyobject:"schema=Post"`
}

type Post struct {
	Title  string
	Body   string
	Author skyobject.Reference `skyobject:"schema=Person"`
}

type Person struct {
	Name string
	Age  uint64
}

// GENERATES:
// pk : 032ffee44b9554cd3350ee16760688b2fb9d0faae7f3534917ff07e971eb36fd6b
// sk : b4f56cab07ea360c16c22ac241738e923b232138b69089fe0134f81a432ffaff
func genKeyPair() (cipher.PubKey, cipher.SecKey) {
	return cipher.GenerateDeterministicKeyPair([]byte("a"))
}

func newRootwalkerClient(t *testing.T, pk cipher.PubKey) (c *Client,
	s *Server) {

	reg := skyobject.NewRegistry()
	reg.Register("Person", Person{})
	reg.Register("Post", Post{})
	reg.Register("Thread", Thread{})
	reg.Register("Board", Board{})

	// feeds
	feeds := []cipher.PubKey{pk}

	var err error
	if c, s, err = newRunningClient(newClientConfig(), reg, feeds); err != nil {
		t.Fatal(err) // fatality
	}

	if c.Subscribe(pk) == false {
		t.Fatal("unable to subscribe")
	}
	return
}

func fillContainer1(c *Container, pk cipher.PubKey,
	sk cipher.SecKey) (r *Root) {

	r, _ = c.NewRoot(pk, sk)

	dynPerson := r.MustDynamic("Person", Person{"Dynamic Beast", 100})
	dynPost := r.MustDynamic("Post",
		Post{"Dynamic Post", "So big.", dynPerson.Object})

	persons := r.SaveArray(
		Person{"Evan", 21},
		Person{"Eric", 23},
		Person{"Jade", 24},
		Person{"Luis", 16},
	)
	posts1 := r.SaveArray(
		Post{"Hi", "Hello?", persons[0]},
		Post{"Bye", "Cya.", persons[0]},
		Post{"Howdy", "Haha.", persons[3]},
	)
	posts2 := r.SaveArray(
		Post{"OK", "Ok then...", persons[1]},
		Post{"What", "Eh what?", persons[2]},
		Post{"Is There?", "Is there really?", persons[3]},
	)
	posts3 := r.SaveArray(
		Post{"Test", "Yeah...", persons[2]},
	)
	threads := r.SaveArray(
		Thread{"Greetings", persons[0], posts1},
		Thread{"Expressions", persons[2], posts2},
		Thread{"Testing", persons[3], posts3},
	)
	r.InjectMany("Board",
		Board{"Test", persons[3], dynPost, threads[2:]},
		Board{"Talk", persons[1], dynPerson, threads[:2]},
	)

	return
}

// getReference obtains a skyobject reference from hex string.
func getReference(s string) (skyobject.Reference, error) {
	h, e := cipher.SHA256FromHex(s)
	return skyobject.Reference(h), e
}

func TestWalker_AdvanceFromRoot(t *testing.T) {
	pk, sk := genKeyPair()

	client, server := newRootwalkerClient(t, pk)
	defer server.Close()
	defer client.Close()

	w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

	board := &Board{}
	e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
		v, e := w.Root().ValueByDynamic(dRef)
		if e != nil {
			return false
		}
		if v.Schema().Name() != "Board" {
			return false
		}
		fv, _ := v.FieldByName("Name")
		s, _ := fv.String()
		return s == "Talk"
	})
	if e != nil {
		t.Error("advance from root failed:", e)
	}
	t.Log(w.String())
}

func TestWalker_AdvanceFromRefsField(t *testing.T) {
	pk, sk := genKeyPair()

	client, server := newRootwalkerClient(t, pk)
	defer server.Close()
	defer client.Close()

	w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

	board := &Board{}
	thread := &Thread{}
	post := &Post{}

	var e error

	e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
		board := &Board{}
		w.DeserializeFromRef(dRef.Object, board)
		return board.Name == "Talk"
	})

	if e != nil {
		t.Error("advance from root to board failed:", e)
	}

	e = w.AdvanceFromRefsField("Threads",
		thread,
		func(i int, ref skyobject.Reference) bool {
			thread := &Thread{}
			w.DeserializeFromRef(ref, thread)
			return thread.Name == "Greetings"
		})

	if e != nil {
		t.Error("advance from board to thread failed:", e)
	}

	t.Log(len(thread.Posts))
	e = w.AdvanceFromRefsField("Posts",
		post,
		func(i int, ref skyobject.Reference) (chosen bool) {
			post := &Post{}
			w.DeserializeFromRef(ref, post)
			return post.Title == "Hi"
		})
	if e != nil {
		t.Error("advance from thread to post failed:", e)
	}
	t.Log("\n", w.String())
}

func TestWalker_GetFromRefsField(t *testing.T) {
	pk, sk := genKeyPair()

	client, server := newRootwalkerClient(t, pk)
	defer server.Close()
	defer client.Close()

	w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

	board := &Board{}
	thread := &Thread{}
	post := &Post{}

	var e error

	e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
		board := &Board{}
		w.DeserializeFromRef(dRef.Object, board)
		return board.Name == "Talk"
	})

	if e != nil {
		t.Error("advance from root to board failed:", e)
	}

	e = w.AdvanceFromRefsField("Threads",
		thread,
		func(i int, ref skyobject.Reference) bool {
			thread := &Thread{}
			w.DeserializeFromRef(ref, thread)
			return thread.Name == "Greetings"
		})

	if e != nil {
		t.Error("advance from board to thread failed:", e)
	}

	e = w.GetFromRefsField("Posts",
		post,
		func(i int, ref skyobject.Reference) bool {
			post := &Post{}
			w.DeserializeFromRef(ref, post)
			return post.Title == "Hi"
		})
	t.Log("\nPost = ", post)
	if e != nil {
		t.Error("get posts from thread failed:", e)
	}
	t.Log("\n", w.String())
}

func TestWalker_AdvanceFromRefField(t *testing.T) {
	pk, sk := genKeyPair()

	client, server := newRootwalkerClient(t, pk)
	defer server.Close()
	defer client.Close()

	w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

	board := &Board{}
	thread := &Thread{}
	person := &Person{}

	var e error

	e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
		board := &Board{}
		w.DeserializeFromRef(dRef.Object, board)
		return board.Name == "Talk"
	})
	if e != nil {
		t.Error("advance from root to board failed:", e)
	}

	e = w.AdvanceFromRefsField("Threads",
		thread,
		func(i int, ref skyobject.Reference) bool {
			thread := &Thread{}
			w.DeserializeFromRef(ref, thread)
			return thread.Name == "Greetings"
		})
	if e != nil {
		t.Error("advance from board to thread failed:", e)
	}

	e = w.AdvanceFromRefField("Creator", person)
	if e != nil {
		t.Error("advance from thread to person failed:", e)
	}
	t.Log("\n", w.String())
}

func TestWalker_GetFromRefField(t *testing.T) {
	pk, sk := genKeyPair()

	client, server := newRootwalkerClient(t, pk)
	defer server.Close()
	defer client.Close()

	w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

	board := &Board{}
	thread := &Thread{}
	person := &Person{}

	var e error

	e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
		board := &Board{}
		w.DeserializeFromRef(dRef.Object, board)
		return board.Name == "Talk"
	})

	if e != nil {
		t.Error("advance from root to board failed:", e)
	}

	e = w.AdvanceFromRefsField("Threads",
		thread,
		func(i int, ref skyobject.Reference) bool {
			thread := &Thread{}
			w.DeserializeFromRef(ref, thread)
			return thread.Name == "Greetings"
		})

	if e != nil {
		t.Error("advance from board to thread failed:", e)
	}

	if e = w.GetFromRefField("Creator", person); e != nil {
		t.Error("advance from thread to person failed:", e)
	}

	t.Log("\nCreator = ", person)
	t.Log("\n", w.String())
}

func TestWalker_AdvanceFromDynamicField(t *testing.T) {
	t.Run("dynamic post", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		post := &Post{}

		var e error

		e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Test"
		})
		if e != nil {
			t.Error("advance from root to board failed:", e)
		}

		e = w.AdvanceFromDynamicField("Featured", post)
		if e != nil {
			t.Error("advance from board to dynamic post failed:", e)
		}
		t.Log("\n", w.String())
	})
	t.Run("dynamic person", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		person := &Person{}

		var e error

		e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root to board failed:", e)
		}

		e = w.AdvanceFromDynamicField("Featured", person)
		if e != nil {
			t.Error("advance from board to dynamic person failed:", e)
		}
		t.Log("\n", w.String())
	})
}

func TestWalker_GetFromDynamicField(t *testing.T) {
	t.Run("dynamic post", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		post := &Post{}

		var e error

		e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Test"
		})
		if e != nil {
			t.Error("advance from root to board failed:", e)
		}

		_, e = w.GetFromDynamicField("Featured", post)
		if e != nil {
			t.Error("advance from board to dynamic post failed:", e)
		}
		t.Log("\nFeatured = ", post)
		t.Log("\n", w.String())
	})
	t.Run("dynamic person", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		person := &Person{}

		var e error

		e = w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root to board failed:", e)
		}

		_, e = w.GetFromDynamicField("Featured", person)
		if e != nil {
			t.Error("advance from board to dynamic person failed:", e)
		}
		t.Log("\nFeatured = ", person)
		t.Log("\n", w.String())
	})
}

func TestWalker_AppendToRefsField(t *testing.T) {
	pk, sk := genKeyPair()

	client, server := newRootwalkerClient(t, pk)
	defer server.Close()
	defer client.Close()

	w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

	board := &Board{}
	e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
		board := &Board{}
		w.DeserializeFromRef(dRef.Object, board)
		return board.Name == "Talk"
	})

	if e != nil {
		t.Error("advance from root failed:", e)
	}

	t.Log("\n Before:", w.String())

	_, e = w.AppendToRefsField("Threads", Thread{Name: "New Thread"})
	if e != nil {
		t.Error("append thread to board failed:", e)
	}
	t.Log("\n After:", w.String())

	thread := &Thread{}
	e = w.AdvanceFromRefsField("Threads",
		thread,
		func(i int, ref skyobject.Reference) bool {
			thread := &Thread{}
			w.DeserializeFromRef(ref, thread)
			return thread.Name == "New Thread"
		})
	if e != nil {
		t.Error("advance from board to thread failed:", e)
	}
	t.Log(w.String())
}

func TestWalker_ReplaceInRefsField(t *testing.T) {
	pk, sk := genKeyPair()

	client, server := newRootwalkerClient(t, pk)
	defer server.Close()
	defer client.Close()

	w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

	board := &Board{}
	e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
		board := &Board{}
		w.DeserializeFromRef(dRef.Object, board)
		return board.Name == "Talk"
	})

	if e != nil {
		t.Error("advance from root failed:", e)
	}

	t.Log("\n Before:", w.String())

	e = w.ReplaceInRefsField("Threads",
		&Thread{Name: "New Thread"},
		func(i int, ref skyobject.Reference) bool {
			thread := &Thread{}
			w.DeserializeFromRef(ref, thread)
			return thread.Name == "Greetings"
		})
	if e != nil {
		t.Error("replace board thread failed:", e)
	}
	t.Log("\n After:", w.String())

	thread := &Thread{}
	e = w.AdvanceFromRefsField("Threads",
		thread,
		func(i int, ref skyobject.Reference) bool {
			thread := &Thread{}
			w.DeserializeFromRef(ref, thread)
			return thread.Name == "New Thread"
		})
	if e != nil {
		t.Error("advance from board to the new thread failed:", e)
	}
	t.Log(w.String())
}

func TestWalker_RemoveInRefsField(t *testing.T) {
	t.Run("depth of 1", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})

		if e != nil {
			t.Error("advance from root failed:", e)
		}

		t.Log("\n Before:", w.String())

		e = w.RemoveInRefsField("Threads",
			func(i int, ref skyobject.Reference) bool {
				thread := &Thread{}
				w.DeserializeFromRef(ref, thread)
				return thread.Name == "Greetings"
			})

		if e != nil {
			t.Error("remove board thread failed:", e)
		}

		t.Log("\n After:", w.String())

		thread := &Thread{}
		e = w.AdvanceFromRefsField("Threads",
			thread,
			func(i int, ref skyobject.Reference) bool {
				thread := &Thread{}
				w.DeserializeFromRef(ref, thread)
				return thread.Name == "Greetings"
			})

		if e != nil {
			t.Log("Removed thread: Greetings")
		} else {
			t.Error("removing thread failed")
		}

		t.Log(w.String())
	})
	t.Run("depth of 2", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})

		if e != nil {
			t.Error("advance from root failed:", e)
		}

		thread := &Thread{}
		e = w.AdvanceFromRefsField("Threads",
			thread,
			func(i int, ref skyobject.Reference) bool {
				thread := &Thread{}
				w.DeserializeFromRef(ref, thread)
				return thread.Name == "Greetings"
			})
		if e != nil {
			t.Error("advance from board to thread failed:", e)
		}
		t.Log("\n Before:", w.String())

		e = w.RemoveInRefsField("Posts",
			func(i int, ref skyobject.Reference) bool {
				post := &Post{}
				w.DeserializeFromRef(ref, post)
				return post.Title == "Bye"
			})

		t.Log("\n After:", w.String())

		e = w.AdvanceFromRefsField("Posts",
			thread,
			func(i int, ref skyobject.Reference) bool {
				post := &Post{}
				w.DeserializeFromRef(ref, post)
				return post.Title == "Bye"
			})

		if e != nil {
			t.Log("Removed post: Bye")
		} else {
			t.Error("removing post failed")
		}
		//Write another test for deeper level
		t.Log(w.String())
	})
}

func TestWalker_ReplaceInRefField(t *testing.T) {
	t.Run("depth of 1", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root failed:", e)
		}
		t.Log(w.String())

		_, e = w.ReplaceInRefField("Creator", Person{"Donald Trump", 70})
		if e != nil {
			t.Error("failed to replace:", e)
		}
		t.Log(w.String())
	})
	t.Run("depth of 2", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root failed:", e)
		}

		thread := &Thread{}
		e = w.AdvanceFromRefsField("Threads",
			thread,
			func(i int, ref skyobject.Reference) bool {
				thread := &Thread{}
				w.DeserializeFromRef(ref, thread)
				return thread.Name == "Greetings"
			})
		if e != nil {
			t.Error("advance from board to thread failed:", e)
		}

		t.Log(w.String())
		{
			p := &Person{}
			data, _ := w.r.Get(thread.Creator)
			encoder.DeserializeRaw(data, p)
			t.Log(p)
		}

		_, e = w.ReplaceInRefField("Creator",
			Person{Name: "Bruce Lee", Age: 77})
		if e != nil {
			t.Error("failed to replace", e)
		}

		t.Log(w.String())
		{
			p := &Person{}
			data, _ := w.r.Get(thread.Creator)
			encoder.DeserializeRaw(data, p)
			t.Log(p)
		}
	})
}

func TestWalker_ReplaceCurrent(t *testing.T) {
	t.Run("depth of 1", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))
		r := w.r
		newPerson, _ := r.Dynamic("Person", Person{"NEW PERSON", 666})
		newPost, _ := r.Dynamic("Post", Post{"NEW", "POST!", newPerson.Object})
		posts := r.SaveArray(
			newPost,
		)
		newThread1, _ := r.Dynamic("Thread",
			Thread{"NEW THREAD!", newPerson.Object, posts})
		newThread2, _ := r.Dynamic("Thread",
			Thread{"ANOTHER THREAD!", newPerson.Object, posts})
		newThreads := r.SaveArray(
			newThread1,
			newThread2,
		)
		newBoard := Board{"NEW!", newPerson.Object, newPost, newThreads}

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		t.Log("Got board", board.Name)
		if e != nil {
			t.Error("advance from root failed:", e)
		}
		t.Log(w.String())
		t.Log("Replacing...")
		e = w.ReplaceCurrent(&newBoard)

		if e != nil {
			t.Error("failed to replace:", e)
		}
		t.Log(w.String())
	})

	t.Run("depth of 2", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		r := w.r
		newPerson, _ := r.Dynamic("Person", Person{"NEW PERSON", 666})
		newPost, _ := r.Dynamic("Post", Post{"NEW", "POST!", newPerson.Object})
		posts := r.SaveArray(
			newPost,
		)
		newThread1 := Thread{"NEW THREAD!", newPerson.Object, posts}

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root failed:", e)
		}
		t.Log("Got board", board.Name)

		thread := &Thread{}
		e = w.AdvanceFromRefsField("Threads",
			thread,
			func(i int, ref skyobject.Reference) bool {
				thread := &Thread{}
				w.DeserializeFromRef(ref, thread)
				return thread.Name == "Greetings"
			})
		if e != nil {
			t.Error("advance from board to thread failed:", e)
		}
		t.Log("Got thread", thread.Name)
		t.Log(w.String())

		e = w.ReplaceCurrent(&newThread1)
		if e != nil {
			t.Error("failed to replace:", e)
		}
		t.Log(w.String())
	})
}

func TestWalker_ReplaceInDynamicField(t *testing.T) {
	t.Run("depth of 1", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root failed:", e)
		}

		t.Log(w.String())
		{
			p := &Person{}
			data, _ := w.r.Get(board.Featured.Object)
			encoder.DeserializeRaw(data, p)
			t.Log(p)
		}

		_, e = w.ReplaceInDynamicField("Featured", "Post",
			Post{Title: "Good Game", Body: "Yeah, this is fun."})
		if e != nil {
			t.Error("replace failed:", e)
		}

		t.Log(w.String())
		{
			p := &Post{}
			data, _ := w.r.Get(board.Featured.Object)
			encoder.DeserializeRaw(data, p)
			t.Log(p)
		}
	})
}

func TestWalker_RemoveCurrent(t *testing.T) {
	t.Run("depth of 1", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		t.Log("Got board", board.Name)
		if e != nil {
			t.Error("advance from root failed:", e)
		}
		t.Log("Size:", len(w.r.Refs()))
		t.Log(w.String())
		t.Log("Removing...")
		e = w.RemoveCurrent()

		if e != nil {
			t.Error("failed to remove:", e)
		}
		t.Log("Size:", len(w.r.Refs()))
		t.Log(w.String())
	})

	t.Run("depth of 2", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root failed:", e)
		}
		t.Log("Got board", board.Name)

		thread := &Thread{}
		e = w.AdvanceFromRefsField("Threads",
			thread,
			func(i int, ref skyobject.Reference) bool {
				thread := &Thread{}
				w.DeserializeFromRef(ref, thread)
				return thread.Name == "Greetings"
			})
		if e != nil {
			t.Error("advance from board to thread failed:", e)
		}
		t.Log("Got thread", thread.Name)
		t.Log(w.String())

		e = w.RemoveCurrent()
		if e != nil {
			t.Error("failed to remove:", e)
		}
		t.Log(w.String())
	})
}

func TestWalker_RemoveByRef(t *testing.T) {
	t.Run("depth of 1", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		t.Log("Got board", board.Name)
		if e != nil {
			t.Error("advance from root failed:", e)
		}
		t.Log("Size:", len(w.r.Refs()))
		t.Log(w.String())
		tRef, e := getReference(
			"28950f97f06483f40662ab6cd841d19db40c2453c22cdf2fec2aab893866ae89")
		t.Log("Removing...", tRef.String(), e)
		e = w.RemoveInRefsByRef("Threads", tRef)

		if e != nil {
			t.Error("failed to remove thread:", e)
		}
		t.Log("Size:", len(w.r.Refs()))
		t.Log(w.String())
	})

	t.Run("depth of 2", func(t *testing.T) {
		pk, sk := genKeyPair()

		client, server := newRootwalkerClient(t, pk)
		defer server.Close()
		defer client.Close()

		w := NewRootWalker(fillContainer1(client.Container(), pk, sk))

		board := &Board{}
		e := w.AdvanceFromRoot(board, func(i int, dRef skyobject.Dynamic) bool {
			board := &Board{}
			w.DeserializeFromRef(dRef.Object, board)
			return board.Name == "Talk"
		})
		if e != nil {
			t.Error("advance from root failed:", e)
		}
		t.Log("Got board", board.Name)

		thread := &Thread{}
		e = w.AdvanceFromRefsField("Threads",
			thread,
			func(i int, ref skyobject.Reference) bool {
				thread := &Thread{}
				w.DeserializeFromRef(ref, thread)
				return thread.Name == "Greetings"
			})
		if e != nil {
			t.Error("advance from board to thread failed:", e)
		}
		t.Log("Got thread", thread.Name)
		t.Log(w.String())

		pRef, e := getReference(
			"bd893dbc1bec38f59a97e103774f279ae1e4f5e8008729dff309ea6c5f169d66")
		t.Log("Removing...", pRef.String(), e)
		e = w.RemoveInRefsByRef("Posts", pRef)
		if e != nil {
			t.Error("failed to remove post:", e)
		}
		t.Log(w.String())
	})
}
