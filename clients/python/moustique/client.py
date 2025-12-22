import socket
import os
import json
import requests
import base64
from datetime import datetime
from typing import Dict, List, Callable, Any, Optional

# Globala variabler
ua = requests.Session()
ua.timeout = 5
sent_cnt = 0
server_ip = ""
server_url = ""
server_port = ""
pickup_intensity = 1
name = "NONAME"

class Moustique:
    def __init__(self, ip: str = "127.0.0.1", port: str = "33335", client_name: str = ""):
        self.pid = os.getpid()
        hostname = socket.gethostname()
        self.name = f"{hostname}-{client_name}-{int(os.urandom(1)[0] % 100)}-{self.pid}" if client_name else f"{hostname}-{int(os.urandom(1)[0] % 100)}-{self.pid}"
        global server_ip, server_url, server_port, name
        server_ip = ip
        server_url = f"http://{ip}"
        server_port = port
        name = self.name
        self.callbacks: Dict[str, List[Callable]] = {}
        self.system_callbacks: Dict[str, Callable] = {}
        self.initialize()

    def initialize(self):
        self.server_ip = server_ip
        self.server_port = server_port
        self.system_callbacks["/server/action/resubscribe"] = self.resubscribe

    def publish(self, topic: str, message: str):
        self.publish_nothread(self.server_ip, self.server_port, topic, message, self.name)

    def putval(self, topic: str, message: str):
            post_url = f"http://{self.server_ip}:{self.server_port}/PUTVAL"
            payload = {
                "valname": Moustique.enc(topic),
                "val": Moustique.enc(message),
                "updated_time": Moustique.enc(str(int(datetime.now().timestamp()))),
                "updated_nicedatetime": Moustique.enc(Moustique.get_nicedatetime()),
                "from": Moustique.enc(self.name)
            }
            try:
                response = ua.put(post_url, data=payload, allow_redirects=True)
                if response.status_code in (200, 308):
                    print(f"Putval till {post_url} lyckades: {response.status_code}")
                else:
                    print(f"Putval oväntad status: {response.status_code} – {response.text}")
            except requests.exceptions.RequestException as e:
                print(f"Fel vid putval till {post_url}: {e}")

    @staticmethod
    def publish_nothread(ip: str, port: str, topic: str, message: str, from_name: str):
        post_url = f"http://{ip}:{port}/POST"
        payload = {
            "topic": Moustique.enc(topic),
            "message": Moustique.enc(message),
            "updated_time": Moustique.enc(str(int(datetime.now().timestamp()))),
            "updated_nicedatetime": Moustique.enc(Moustique.get_nicedatetime()),
            "from": Moustique.enc(from_name)
        }
        try:
            response = ua.post(post_url, data=payload)
            response.raise_for_status()
            print(f"Publish till {post_url} lyckades: {response.status_code}")
        except requests.exceptions.RequestException as e:
            print(f"Fel vid publish till {post_url}: {e}")


    def subscribe(self, topic: str, callback: Callable, consumer: str = ""):
        payload = {"topic": Moustique.enc(topic), "client": Moustique.enc(self.name)}
        try:
            response = ua.post(f"{server_url}:{server_port}/SUBSCRIBE", data=payload)
            response.raise_for_status()
            print(f"{self.name} subscrbar pa {topic}")
            if topic not in self.callbacks:
                self.callbacks[topic] = []
            if callback not in self.callbacks[topic]:
                self.callbacks[topic].append(callback)
            else:
                print(f"Hittade samma callback för ämnet {topic}!")
        except requests.exceptions.RequestException as e:
            print(f"Fel vid subscribe till {topic}: {e}")

    def resubscribe(self, *args):
        log_topic = "/mushroom/logs/moustique_lib/INFO"
        if self.callbacks:
            self.publish(log_topic, f"{self.name} Resubscribing all subscriptions")
        for topic in list(self.callbacks.keys()):
            print(f"Resubscribing {topic} ({self.name})")
            payload = {"topic": Moustique.enc(topic), "client": Moustique.enc(self.name)}
            try:
                response = ua.post(f"{server_url}:{server_port}/SUBSCRIBE", data=payload, timeout=5)
                response.raise_for_status()
            except requests.exceptions.RequestException as e:
                print(f"Fel vid resubscribe för {topic}: {e}")
        self.publish(log_topic, f"{self.name} Resubscribed all subscriptions")

    def tick(self, consumer: str = ""):
        self.pickup()
        return 1

    def get_val(self, valname: str) -> Optional[dict]:
        post_url = f"http://{self.server_ip}:{self.server_port}/GETVAL"
        payload = {"client": Moustique.enc(self.name), "topic": Moustique.enc(valname)}
        try:
            response = ua.post(post_url, data=payload)
            response.raise_for_status()
            decoded_text = Moustique.dec(response.text.strip())
            return json.loads(decoded_text) if decoded_text else None
        except (requests.exceptions.RequestException, json.JSONDecodeError, base64.binascii.Error) as e:
            print(f"Fel vid get_val för {valname}: {e}")
            return None

    @staticmethod
    def getval(ip: str, port: str, valname: str) -> Optional[dict]:
        post_url = f"http://{ip}:{port}/GETVAL"
        payload = {"client": Moustique.enc(name), "topic": Moustique.enc(valname)}
        try:
            response = ua.post(post_url, data=payload)
            response.raise_for_status()
            decoded_text = Moustique.dec(response.text.strip())
            return json.loads(decoded_text) if decoded_text else None
        except (requests.exceptions.RequestException, json.JSONDecodeError, base64.binascii.Error) as e:
            print(f"Fel vid getval för {valname}: {e}")
            return None

    @staticmethod
    def pitval(ip: str, port: str, topic: str, message: str, from_name: str):
        Moustique.publish_nothread_put(ip, port, topic, message, from_name)

    @staticmethod
    def getversion(ip: str, port: str, pwd: str) -> Optional[dict]:
        return Moustique.get(ip, port, pwd, "VERSION")

    @staticmethod
    def get(ip: str, port: str, pwd: str, endpoint: str, retries: int = 0) -> Optional[dict]:
        post_url = f"http://{ip}:{port}/{endpoint}"
        payload = {"client": Moustique.enc(name), "pwd": Moustique.enc(pwd)}
        try:
            response = ua.post(post_url, data=payload)
            response.raise_for_status()
            decoded_text = Moustique.dec(response.text.strip())
            return json.loads(decoded_text) if decoded_text else None
        except (requests.exceptions.RequestException, json.JSONDecodeError, base64.binascii.Error) as e:
            if hasattr(e, 'response') and e.response.status_code == 401:
                print("Vänligen ange rätt pwd.")
            elif retries < 5:
                print(f"Fel vid get ({endpoint}), försök {retries + 1}/5: {e}")
                return Moustique.get(ip, port, pwd, endpoint, retries + 1)
            else:
                print(f"Misslyckades med get ({endpoint}) efter 5 försök: {e}")
            return None

    def pickup(self):
        payload = {"client": Moustique.enc(self.name)}
        post_url = f"{server_url}:{server_port}/PICKUP"
        try:
            response = ua.post(post_url, data=payload)
            response.raise_for_status()
            decoded_text = Moustique.dec(response.text.strip())
            data = json.loads(decoded_text) if decoded_text else {}
            if data:
                for topic, messages in data.items():
                    for message in messages:
                        topic_callbacks = self.callbacks.get(topic, [])
                        for callback in topic_callbacks:
                            callback(message["topic"], message["message"], message["from"])
                        system_callback = self.system_callbacks.get(topic)
                        if system_callback and not topic_callbacks:
                            system_callback(message["topic"], message["message"])
        except (requests.exceptions.RequestException, json.JSONDecodeError, base64.binascii.Error) as e:
            print(f"Fel vid pickup för {post_url}: {e} (rådata: {response.text if 'response' in locals() else 'ingen respons'})")

    @staticmethod
    def get_nicedatetime() -> str:
        now = datetime.now()
        return now.strftime("%Y-%m-%d %H:%M:%S")

    def get_client_name(self) -> str:
        return self.name

    @staticmethod
    def enc(plaintext: str) -> str:
        if plaintext:
            encoded = base64.b64encode(plaintext.encode()).decode()
            return encoded.translate(str.maketrans(
                'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz',
                'NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm'))
        return ""

    @staticmethod
    def dec(encoded: str) -> str:
        if not encoded:
            return ""
        decoded = encoded.translate(str.maketrans(
            'NOPQRSTUVWXYZABCDEFGHIJKLMnopqrstuvwxyzabcdefghijklm',
            'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'))
        return base64.b64decode(decoded).decode()

# Hjälpfunktioner
def getversion(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "VERSION")

def getfileversion(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "FILEVERSION")

def getstats(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "STATS")

def getclients(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "CLIENTS")

def getposters(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "POSTERS")

def gettopics(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "TOPICS")

def getpeerhosts(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "PEERHOSTS")

def getcrooks(ip: str, port: str, pwd: str) -> Optional[dict]:
    return Moustique.get(ip, port, pwd, "CROOKS")

# Testklient
import time

def message_callback(topic: str, message: str, from_name: str):
    print(f"Mottaget meddelande på ämne '{topic}': {message} från {from_name}")

def main():
    client = Moustique(ip="127.0.0.1", port="33335", client_name="TestClient")
    print(f"Klientnamn: {client.get_client_name()}")

    try:
        print("\nTest 1: Publicerar ett meddelande...")
        client.publish("/test/topic", "Hej, detta är ett testmeddelande!")
        time.sleep(1)

        print("\nTest 2: Sätter ett värde med putval...")
        client.putval("127.0.0.1", "33335", "/test/value", "42", client.get_client_name())
        time.sleep(1)

        print("\nTest 3: Hämtar ett värde med getval...")
        value = client.get_val("/test/value")
        print(f"Hämtat värde: {value}")

        print("\nTest 4: Prenumererar på ett ämne...")
        client.subscribe("/test/topic", message_callback)
        print("Publicerar ett meddelande till prenumererat ämne...")
        client.publish("/test/topic", "Detta borde trigga callback!")
        
        print("Kör pickup i 5 sekunder för att fånga meddelanden...")
        end_time = time.time() + 5
        while time.time() < end_time:
            client.tick()
            time.sleep(0.5)

        print("\nTest 5: Testar resubscribe...")
        client.resubscribe()

        print("\nTest 6: Hämtar serverinformation...")
        version = getversion("127.0.0.1", "33335", "1Delmataren")  # Ersätt "password" med rätt lösenord
        print(f"Serverversion: {version}")
        stats = getstats("127.0.0.1", "33335", "1Delmataren")
        print(f"Serverstatistik: {stats}")
    except requests.exceptions.ConnectionError as e:
        print(f"Anslutningsfel: {e}. Är servern igång på 127.0.0.1:33335?")
    except Exception as e:
        print(f"Ett oväntat fel uppstod: {e}")

if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        print("\nAvslutar klienten...")
    except Exception as e:
        print(f"Ett fel uppstod: {e}")
