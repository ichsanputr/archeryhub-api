import os
import requests
from concurrent.futures import ThreadPoolExecutor

# Configuration
SAVE_DIR = "media"
MALE_COUNT = 100
FEMALE_COUNT = 80
BASE_URL = "https://randomuser.me/api/portraits"

# Ensure directory exists
if not os.path.exists(SAVE_DIR):
    os.makedirs(SAVE_DIR)

def download_image(args):
    gender, i = args
    url = f"{BASE_URL}/{'men' if gender == 'male' else 'women'}/{i}.jpg"
    filename = f"{gender}_{i}.jpg"
    filepath = os.path.join(SAVE_DIR, filename)
    
    if os.path.exists(filepath):
        return filename

    try:
        response = requests.get(url, timeout=10)
        if response.status_code == 200:
            with open(filepath, 'wb') as f:
                f.write(response.content)
            return filename
        else:
            print(f"Failed to download {url}: {response.status_code}")
    except Exception as e:
        print(f"Error downloading {url}: {e}")
    return None

def main():
    tasks = []
    for i in range(1, MALE_COUNT + 1):
        tasks.append(("male", i))
    for i in range(1, FEMALE_COUNT + 1):
        tasks.append(("female", i))

    print(f"Downloading {len(tasks)} images in parallel...")
    with ThreadPoolExecutor(max_workers=10) as executor:
        results = list(executor.map(download_image, tasks))

    downloaded = [r for r in results if r]
    print(f"Download complete. Total downloaded: {len(downloaded)}")

if __name__ == "__main__":
    main()
