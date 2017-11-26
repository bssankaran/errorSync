# errorSync
A go library to which helps to wait for many goroutines, like WaitGroup and additionally, returns errors if any.
A call to the Wait() method of ErrorWaitGroup waits until all the related goroutines are completed (calls Done()) or if any one of them sends error(calls SendError()).
