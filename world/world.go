package world

type Config struct {

}

/*
DESIGN MUSINGS: So I (not specifically due to performance) will use a Structure-of-
Arrays (SoA) to keep track of which Gorgon IDs map to which goroutines and so that
I can cancel them (aka kill them) as required.

The profiler package I wrote will now, in all probability, go to waste, but well, 
at least I learned something new,
*/

type World struct {
	/*
	As we create gorgons to execute self-mutating code, we will assign
	incremental indices to each of the gorgons. 
	*/
	signals []chan<- string
	// readChannels will be used to extract information from each gorgon
	readChannels []<-chan string // this type can be changed later
}

