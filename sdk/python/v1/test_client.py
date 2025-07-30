import unittest
unittest.TestLoader.sortTestMethodsUsing = None

from .client import Client, Config


class TestClient(unittest.TestCase):
    def test_push(self):
        pass

    def test_pull(self):
        pass

    def test_lookup(self):
        pass

    def test_publish(self):
        pass

    def test_list(self):
        pass

    def test_search(self):
        pass

    def test_unpublish(self):
        pass

    def test_delete(self):
        pass

if __name__ == "__main__":
    unittest.main()
