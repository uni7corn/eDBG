export NDK_ROOT=/home/kali/android-ndk-r25c

export PATH=$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/bin:$PATH
# export PATH=$NDK_ROOT/toolchains/llvm/prebuilt/darwin-x86_64/bin:$PATH

make clean && make
