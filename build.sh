mkdir -p build
rm -rf build/*
NDK=$HOME/Android/Sdk/ndk/29.0.13113456
API=30

## For ARM64 (64-bit)
# TARGET=aarch64-linux-android
# TOOLCHAIN=$NDK/toolchains/llvm/prebuilt/linux-x86_64
# CC="$TOOLCHAIN/bin/clang --target=$TARGET$API" CXX="$TOOLCHAIN/bin/clang --target=$TARGET$API" GOOS=android GOARCH=arm64 CGO_ENABLED=1 go build -ldflags="-s -w" -o build/mouse main.go

# Or for ARMv7 (32-bit)
 TARGET=armv7a-linux-androideabi
 TOOLCHAIN=$NDK/toolchains/llvm/prebuilt/linux-x86_64
 CC="$TOOLCHAIN/bin/clang --target=$TARGET$API" CXX="$TOOLCHAIN/bin/clang --target=$TARGET$API" GOOS=android GOARCH=arm GOARM=7 CGO_ENABLED=1 go build -ldflags="-s -w" -o build/mouse main.go

cd build
upx mouse
cp -r ../properties/* .
zip -r -FS goFlipMouse.zip .