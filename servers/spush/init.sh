#!/bin/bash
cd tools
find . -name "*.sh" | xargs -I{} dos2unix {}
find . -name "*.sh" | xargs -I{} chmod u+x {}

find . -name "*.exp" | xargs -I{} dos2unix {}
find . -name "*.exp" | xargs -I{} chmod u+x {}
