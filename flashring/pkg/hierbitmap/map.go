package hierbitmap

type Bitmap64 [64]uint64

type Level3 struct {
	Leafs [64]interface{}
	Sum   Bitmap64
}

type Level2 struct {
	Nodes [64]Level3
	Sum   Bitmap64
}

type Level1 struct {
	Nodes [64]Level2
	Sum   Bitmap64
}

type Level0 struct {
	Level1 [64]Level1
	Sum    Bitmap64
}
