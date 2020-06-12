#!/bin/bash
#you may execute this file firstly!
echo "adapting..."
find . -name "*.sh" | xargs -i{} dos2unix {}
find . -name "*.py" | xargs -i{} dos2unix {}
echo "done"
