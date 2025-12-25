"""
Moustique Python Client - Multi-tenant version (Fixed)
Supports optional username/password authentication
"""

import requests
import base64
import codecs
import json
from typing import Callable, Optional, Dict, Any, List

def rot13(text: str) -> str:
    """Apply ROT13 cipher"""
    result = []
    for char in text:
        if 'a' <= char <= 'z':
            result.append(chr((ord(char) - ord('a') + 13) % 26 + ord('a')))
        elif 'A' <= char <= 'Z':
            result.append(chr((ord(char) - ord('A') + 13) % 26 + ord('A')))
        else:
            result.append(char)
    return ''.join(result)

def encode_rot13_base64(text: str) -> str:
    """Encode text with ROT13 and Base64 (ROT13 first, then Base64)"""
    rot13_text = rot13(text)
    b64 = base64.b64encode(rot13_text.encode('utf-8')).decode('ascii')
    return b64

def decode_rot13_base64(encoded: str) -> str:
    """Decode Base64 and ROT13 (Base64 first, then ROT13)"""
    try:
        decoded = base64.b64decode(encoded).decode('utf-8')
        return rot13(decoded)
    except Exception as e:
        print(f"Decode error: {e}")
        return encoded

class Moustique:
    def __init__(self, ip: str, port: str, client_name: str, 
                 username: Optional[str] = None, password: Optional[str] = None, 
                 timeout: int = 5):
        """
        Initialize Moustique client
        
        Args:
            ip: Server IP address
            port: Server port
            client_name: Unique client identifier
            username: Username for authentication (optional if public access enabled)
            password: Password for authentication (optional if public access enabled)
            timeout: Request timeout in seconds
        """
        self.ip = ip
        self.port = port
        self.client_name = client_name
        self.username = username
        self.password = password
        self.timeout = timeout
        self.base_url = f"http://{ip}:{port}"
        self.callbacks = {}
        self.session = requests.Session()  # Reuse connections
        
    def _make_request(self, endpoint: str, params: Dict[str, str]) -> Any:
        """Make HTTP request with optional authentication"""
        # Add authentication if provided
        if self.username and self.password:
            params['username'] = self.username
            params['password'] = self.password
        
        # Encode all parameters
        encoded_params = {}
        for key, value in params.items():
            encoded_params[key] = encode_rot13_base64(str(value))
        
        try:
            response = self.session.post(
                f"{self.base_url}/{endpoint}",
                data=encoded_params,
                timeout=self.timeout
            )
            
            if response.status_code == 401:
                raise Exception("Authentication failed: Invalid username or password")
            
            if response.status_code == 404:
                raise Exception(f"Endpoint not found: /{endpoint}")
                
            response.raise_for_status()
            
            # Handle empty response (e.g., for POST, SUBSCRIBE)
            if not response.text or len(response.text.strip()) == 0:
                return None
            
            # Decode response
            try:
                decoded = decode_rot13_base64(response.text)
                return json.loads(decoded)
            except json.JSONDecodeError as e:
                print(f"JSON decode error: {e}")
                print(f"Decoded text: {decoded}")
                return None
            
        except requests.exceptions.Timeout:
            raise Exception(f"Request timeout after {self.timeout} seconds")
        except requests.exceptions.ConnectionError:
            raise Exception(f"Connection error: Could not connect to {self.base_url}")
        except requests.exceptions.RequestException as e:
            raise Exception(f"Request failed: {e}")
    
    def publish(self, topic: str, message: str, from_name: Optional[str] = None) -> None:
        """Publish a message to a topic"""
        params = {
            'topic': topic,
            'message': message,
            'from': from_name or self.client_name
        }
        self._make_request('POST', params)
    
    def subscribe(self, topic: str, callback: Callable[[str, str, str], None]) -> None:
        """Subscribe to a topic with a callback function"""
        params = {
            'topic': topic,
            'client': self.client_name
        }
        self._make_request('SUBSCRIBE', params)
        self.callbacks[topic] = callback
    
    def putval(self, key: str, value: str, from_name: Optional[str] = None) -> None:
        """Store a key-value pair"""
        params = {
            'valname': key,
            'val': value,
            'from': from_name or self.client_name
        }
        self._make_request('PUTVAL', params)
    
    def get_val(self, key: str) -> Any:
        """Retrieve a value by key"""
        params = {'topic': key}
        result = self._make_request('GETVAL', params)
        if result and isinstance(result, dict):
            return result.get('val') or result.get('message')
        return result
    
    def pickup(self) -> List[Dict]:
        """Poll for new messages and trigger callbacks"""
        params = {'client': self.client_name}
        result = self._make_request('PICKUP', params)

        all_messages = []

        # Handle both old format (list) and new format (dict of topic: [messages])
        if result:
            if isinstance(result, dict):
                # New format: {"/topic1": [msg1, msg2], "/topic2": [msg3]}
                for topic, messages in result.items():
                    if isinstance(messages, list):
                        for msg in messages:
                            topic_name = msg.get('topic', topic)
                            message = msg.get('message', '')
                            from_name = msg.get('from', '')

                            # Trigger callback if subscribed to this topic
                            if topic in self.callbacks:
                                try:
                                    self.callbacks[topic](topic_name, message, from_name)
                                except Exception as e:
                                    print(f"Error in callback for topic '{topic}': {e}")

                            all_messages.append(msg)
            elif isinstance(result, list):
                # Old format: [msg1, msg2, msg3]
                for msg in result:
                    topic = msg.get('topic', '')
                    message = msg.get('message', '')
                    from_name = msg.get('from', '')

                    # Trigger callback if subscribed to this topic
                    if topic in self.callbacks:
                        try:
                            self.callbacks[topic](topic, message, from_name)
                        except Exception as e:
                            print(f"Error in callback for topic '{topic}': {e}")

                all_messages = result

        return all_messages
    
    def tick(self) -> List[Dict]:
        """Alias for pickup()"""
        return self.pickup()
    
    def get_client_name(self) -> str:
        """Get the client name"""
        return self.client_name
    
    def resubscribe(self) -> None:
        """Resubscribe to all topics"""
        for topic in list(self.callbacks.keys()):
            params = {
                'topic': topic,
                'client': self.client_name
            }
            try:
                self._make_request('SUBSCRIBE', params)
            except Exception as e:
                print(f"Failed to resubscribe to '{topic}': {e}")


# Helper functions for server information

def getversion(ip: str, port: str) -> str:
    """Get server version (no authentication required)"""
    try:
        url = f"http://{ip}:{port}/VERSION"
        response = requests.get(url, timeout=5)
        response.raise_for_status()
        
        if response.text:
            decoded = decode_rot13_base64(response.text)
            # Remove quotes if present
            return decoded.strip('"\'')
        return "unknown"
    except Exception as e:
        print(f"Failed to get version: {e}")
        return "unknown"

def getstats(ip: str, port: str, username: Optional[str] = None, password: Optional[str] = None) -> Dict:
    """Get server statistics (requires authentication if not public)"""
    try:
        params = {}
        if username and password:
            params = {
                'username': encode_rot13_base64(username),
                'password': encode_rot13_base64(password)
            }
        
        url = f"http://{ip}:{port}/STATS"
        response = requests.post(url, data=params, timeout=5)
        
        if response.status_code == 401:
            raise Exception("Authentication failed - username and password required")
        
        response.raise_for_status()
        
        if response.text:
            decoded = decode_rot13_base64(response.text)
            return json.loads(decoded)
        return {}
    except Exception as e:
        print(f"Failed to get stats: {e}")
        return {}

def getclients(ip: str, port: str, username: Optional[str] = None, password: Optional[str] = None) -> list:
    """Get active clients (requires authentication if not public)"""
    try:
        params = {}
        if username and password:
            params = {
                'username': encode_rot13_base64(username),
                'password': encode_rot13_base64(password)
            }
        
        url = f"http://{ip}:{port}/CLIENTS"
        response = requests.post(url, data=params, timeout=5)
        
        if response.status_code == 401:
            raise Exception("Authentication failed - username and password required")
        
        response.raise_for_status()
        
        if response.text:
            decoded = decode_rot13_base64(response.text)
            return json.loads(decoded)
        return []
    except Exception as e:
        print(f"Failed to get clients: {e}")
        return []

def gettopics(ip: str, port: str, username: Optional[str] = None, password: Optional[str] = None) -> list:
    """Get all topics (requires authentication if not public)"""
    try:
        params = {}
        if username and password:
            params = {
                'username': encode_rot13_base64(username),
                'password': encode_rot13_base64(password)
            }

        url = f"http://{ip}:{port}/TOPICS"
        response = requests.post(url, data=params, timeout=5)

        if response.status_code == 401:
            raise Exception("Authentication failed - username and password required")

        response.raise_for_status()

        if response.text:
            decoded = decode_rot13_base64(response.text)
            return json.loads(decoded)
        return []
    except Exception as e:
        print(f"Failed to get topics: {e}")
        return []

def getposters(ip: str, port: str, username: Optional[str] = None, password: Optional[str] = None) -> list:
    """Get all posters (requires authentication if not public)"""
    try:
        params = {}
        if username and password:
            params = {
                'username': encode_rot13_base64(username),
                'password': encode_rot13_base64(password)
            }

        url = f"http://{ip}:{port}/POSTERS"
        response = requests.post(url, data=params, timeout=5)

        if response.status_code == 401:
            raise Exception("Authentication failed - username and password required")

        response.raise_for_status()

        if response.text:
            decoded = decode_rot13_base64(response.text)
            return json.loads(decoded)
        return []
    except Exception as e:
        print(f"Failed to get posters: {e}")
        return []

def getpeerhosts(ip: str, port: str, username: Optional[str] = None, password: Optional[str] = None) -> list:
    """Get all peer hosts (requires authentication if not public)"""
    try:
        params = {}
        if username and password:
            params = {
                'username': encode_rot13_base64(username),
                'password': encode_rot13_base64(password)
            }

        url = f"http://{ip}:{port}/PEERHOSTS"
        response = requests.post(url, data=params, timeout=5)

        if response.status_code == 401:
            raise Exception("Authentication failed - username and password required")

        response.raise_for_status()

        if response.text:
            decoded = decode_rot13_base64(response.text)
            return json.loads(decoded)
        return []
    except Exception as e:
        print(f"Failed to get peer hosts: {e}")
        return []

def getcrooks(ip: str, port: str, username: Optional[str] = None, password: Optional[str] = None) -> list:
    """Get all crooks/banned IPs (requires authentication if not public)"""
    try:
        params = {}
        if username and password:
            params = {
                'username': encode_rot13_base64(username),
                'password': encode_rot13_base64(password)
            }

        url = f"http://{ip}:{port}/CROOKS"
        response = requests.post(url, data=params, timeout=5)

        if response.status_code == 401:
            raise Exception("Authentication failed - username and password required")

        response.raise_for_status()

        if response.text:
            decoded = decode_rot13_base64(response.text)
            return json.loads(decoded)
        return []
    except Exception as e:
        print(f"Failed to get crooks: {e}")
        return []
