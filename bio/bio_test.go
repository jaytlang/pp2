package bio

// Test the bio interface, Binit/Bget/Bpush/Brelse/Brenew,
// in complete isolation assuming that the disk works.

// Partitions:
// Bget:
//	-> Nr
//		-> Corresponds to empty block, doesn't
//		-> Is very large, is very small
// Bpush:
//	-> b
//		-> Data has been changed, hasn't
//		-> Data was empty, isn't now
//		-> Block number is very large, is very small
// 		-> Block lock is held, isn't (=FAILURE)
// Brenew:
//	-> b
//		-> Block lock is held, isn't (=FAILURE)
//		-> Block number is very large, is very small
//		-> Block data has been changed, hasn't
// Brelse:
//	-> b
//		-> Block lock is held, isn't (=FAILURE)
//		-> Block number is very large, is very small
//		-> Block data does not persist independent of Bpush
