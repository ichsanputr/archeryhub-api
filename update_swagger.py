import os
import re

directories = [
    r"c:\E\ichsan\startup\app\archeris.net\api\handler\mobile",
    r"c:\E\ichsan\startup\app\archeris.net\api\handler"
]

for directory in directories:
    if not os.path.exists(directory):
        continue
    for filename in os.listdir(directory):
        if filename.endswith(".go"):
            filepath = os.path.join(directory, filename)
            with open(filepath, 'r', encoding='utf-8') as f:
                content = f.read()
            
            # 1. Remove "Mobile - " from Tags (any number of spaces)
            new_content = re.sub(r"@Tags\s+Mobile - ", "@Tags         ", content)
            
            # 2. Remove "/mobile" from Router (any number of spaces)
            # Match @Router followed by spaces, then /mobile
            new_content = re.sub(r"@Router\s+/mobile", "@Router       ", new_content)
            
            if new_content != content:
                with open(filepath, 'w', encoding='utf-8') as f:
                    f.write(new_content)
                print(f"Updated {filename}")

# Also check main.go (redundant but safe)
main_path = r"c:\E\ichsan\startup\app\archeris.net\api\main.go"
with open(main_path, 'r', encoding='utf-8') as f:
    content = f.read()
new_content = re.sub(r"@Tags\s+Mobile - ", "@Tags         ", content)
new_content = re.sub(r"@Router\s+/mobile", "@Router       ", new_content)
if new_content != content:
    with open(main_path, 'w', encoding='utf-8') as f:
        f.write(new_content)
    print(f"Updated main.go")

