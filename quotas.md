This needs to turn into a document that contributors understand

TODOs:
- push Etc: done for requiring mem, should release mem when register is cleared?
- strings: streamline requiring mem

Implementations Guidelines:
- in a loop require cpu proportional to the number of iterations in the loop
- when creating a Value require memory
- when creating a slice of values require memory
- when calling a Go standard library function you may want to require memory /
  cpu depending on the characteristics of this function

Testing guidelines
- write *.quotas.lua test file, using quota.rcall to check that memory and cpu
  are accounted for.