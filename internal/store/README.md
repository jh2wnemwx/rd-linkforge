# store package

## Defines the following new errors
```
var (
	ErrEmptyStore  = errors.New("store: empty store")
	ErrInvalidLink = errors.New("store: ID and Target cannot be empty")
	ErrDuplicateID = errors.New("store: duplicate ID")
)
```

## Implements the following functions
```
func New() *Store                               //Initialize an empty store  
func Add(s *Store, l link.Link) error           //Add new item to store  
func Get(s *Store, id uint64) (link.Link, bool) //Get an item from store
func All(s *Store) []link.Link                  //Sort items by ID ascending
func Count(s *Store) int                        //count all elements in store
```

## Implementation notes
There is one deviation from the initial task: since we have already covered pointers on multiple occasions, I opted to pass Store by reference rather than by value, which would have incurred a significant performance penalty.