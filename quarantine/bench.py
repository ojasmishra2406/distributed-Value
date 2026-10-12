import time
import requests
import statistics

def bench():
    url = "http://localhost:8000/predict/user_0"
    latencies = []
    
    print("Sending 30 requests to the prediction endpoint...")
    for _ in range(30):
        start = time.time()
        resp = requests.get(url)
        latencies.append((time.time() - start) * 1000) # ms
        if resp.status_code != 200:
            print("Error:", resp.text)
            
    latencies.sort()
    p50 = latencies[len(latencies) // 2]
    print(f"Executed 30 queries.")
    print(f"p50 End-to-End Latency (KV Fetch + ML Inference): {p50:.2f} ms")

if __name__ == "__main__":
    bench()
