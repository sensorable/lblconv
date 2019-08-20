github.com/ryszard/tfutils was included here because the original repository contains
github.com/tensorflow/tensorflow as a git submodule. This is not actually a dependency, but `go get`
pulls it nonetheless, which is not what we want given the size of the tensorflow repository.
