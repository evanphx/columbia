# Project Columbia

## What is this?

Imagine for a moment that docker images were not tar files full of amd64 ELF binaries.

But instead were tar files full of WebAssembly binaries, compiled against a custom libc
that used a strict syscall boundary just like a real kernel.

That is what Project Columbia is.

## Does it work??

It does! A sample busybox image will be uploaded shortly that folks can try.

## So I can use this instead of docker right now?

HAHAHAH. (cough)

No. Eventually maybe! It doesn't do 99% of what docker does yet. The aim right now
is just to flesh out the syscall boundary layer.

## Any big sticking points?

LLVM/Clang's linker, `lld`, doesn't yet support dynamically linked webassembly. That's
going to be a limiting factor pretty early for anything using `dlopen`. Hopefully
that will get solved eventually.


