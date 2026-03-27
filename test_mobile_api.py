import requests
import json
import os

# New Base URL including /mobile as per request
BASE_URL = "http://localhost:8001/api/v1/mobile"

# Test Credentials (based on current DB data)
ARCHER_EMAIL = "stewie4king@gmail.com"
ARCHER_PASSWORD = "12345"
ORG_EMAIL = "sekretariat@perpanidki.id"
ORG_PASSWORD = "password123"
SELLER_EMAIL = "seller@panahan.com"
SELLER_PASSWORD = "12345"
SK_CODE = "H7K9P"

def test_endpoint(name, method, path, payload=None, token=None):
    url = f"{BASE_URL}{path}"
    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    
    print(f"Testing {name}: {method} {path}...", end=" ", flush=True)
    try:
        if method == "GET":
            response = requests.get(url, headers=headers)
        elif method == "POST":
            response = requests.post(url, json=payload, headers=headers)
        elif method == "PUT":
            response = requests.put(url, json=payload, headers=headers)
        
        if response.status_code == 200 or response.status_code == 201:
            print("[SUCCESS]")
            return response.json()
        else:
            print(f"[FAILED] - Status: {response.status_code}")
            # print(f"Response: {response.text}")
            return None
    except Exception as e:
        print(f"[ERROR] - {str(e)}")
        return None

def main():
    print("=== ArcheryHub Mobile API Test Script ===\n")

    # 1. Public Hello
    test_endpoint("Mobile Hello", "GET", "/hello")

    # 2. Login Tests
    print("\n--- Login Tests ---")
    archer_login = test_endpoint("Archer Login", "POST", "/auth/login", {"email": ARCHER_EMAIL, "password": ARCHER_PASSWORD})
    org_login = test_endpoint("Org Login", "POST", "/auth/organization/login", {"email": ORG_EMAIL, "password": ORG_PASSWORD})
    seller_login = test_endpoint("Seller Login", "POST", "/auth/seller/login", {"email": SELLER_EMAIL, "password": SELLER_PASSWORD})
    sk_login = test_endpoint("Scorekeeper Login", "POST", "/auth/scorekeeper/login", {"code": SK_CODE})

    archer_token = archer_login.get("token") if archer_login else None
    org_token = org_login.get("token") if org_login else None
    seller_token = seller_login.get("token") if seller_login else None
    sk_token = sk_login.get("token") if sk_login else None

    # 3. Public Mobile Routes
    print("\n--- Public Mobile Routes ---")
    events = test_endpoint("List Events", "GET", "/events")
    event_slug = events['events'][0]['slug'] if events and events.get('events') else "seleksi-popda-kabsleman-2026-7247378c"
    test_endpoint("Event Detail", "GET", f"/events/{event_slug}")
    
    news_res = test_endpoint("List News", "GET", "/news")
    if news_res and news_res.get('news') and len(news_res['news']) > 0:
        news_id = news_res['news'][0]['id']
        test_endpoint("News Detail", "GET", f"/news/{news_id}")

    test_endpoint("Marketplace Products", "GET", "/marketplace/products")

    # 4. Archer Specific Routes (requires archer_token)
    if archer_token:
        print("\n--- Archer Authenticated Routes ---")
        test_endpoint("Archer My Events", "GET", "/archer/events", token=archer_token)
        test_endpoint("Archer Cart", "GET", "/archer/cart", token=archer_token)
        test_endpoint("Archer Orders", "GET", "/archer/orders", token=archer_token)
        
        # Test Event Registration Specifics
        event_uuid = events['events'][0]['uuid'] if events and events.get('events') else "event-uuid"
        test_endpoint("Archer Event Registration Detail", "GET", f"/archer/events/{event_uuid}/registration", token=archer_token)
        test_endpoint("Archer Event QR", "GET", f"/archer/events/{event_uuid}/qr", token=archer_token)
        
        # Chat
        test_endpoint("Chat Conversations", "GET", "/chat/conversations", token=archer_token)
        test_endpoint("Chat Unread Count", "GET", "/chat/unread", token=archer_token)

    # 5. Organization Specific Routes (requires org_token)
    if org_token:
        print("\n--- Organization Authenticated Routes ---")
        test_endpoint("Organization Me", "GET", "/organization/me", token=org_token)
        test_endpoint("Organization My Events", "GET", "/organization/events", token=org_token)

    # 6. Seller Specific Routes (requires seller_token)
    if seller_token:
        print("\n--- Seller Authenticated Routes ---")
        test_endpoint("Seller Me", "GET", "/seller/me", token=seller_token)
        test_endpoint("Seller Products", "GET", "/seller/products", token=seller_token)

    # 7. Scorekeeper Specific Routes (requires sk_token)
    if sk_token:
        print("\n--- Scorekeeper Authenticated Routes ---")
        test_endpoint("Scorekeeper Me", "GET", "/scorekeeper/me", token=sk_token)
        test_endpoint("Scorekeeper Events", "GET", "/scorekeeper/events", token=sk_token)
        
        # Scan / Sessions
        test_endpoint("Scan Target (dummy)", "GET", "/scan?code=TARGET-001", token=sk_token)
        test_endpoint("Session Boards", "GET", "/sessions/boards", token=sk_token)

    print("\n=== Test Finished ===")

if __name__ == "__main__":
    main()
