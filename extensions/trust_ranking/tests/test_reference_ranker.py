import unittest

from extensions.trust_ranking.reference_ranker import rank_agents


class TestReferenceRanker(unittest.TestCase):
    def setUp(self):
        # Minimal "good" agent: complete + verified + fresh + clean behavior
        self.good_agent = {
            "id": "agent_good",
            "name": "Good Agent",
            "url": "https://good.example",
            "capabilities": ["book"],
            "contact": "ops@good.example",
            "updated_at": "2025-12-28",
            "domain_verified": True,
            "key_present": True,
            "handshake_fail_ratio": 0.0,
            "rate_limit_violations": 0,
            "complaint_flags": 0,
        }

        # Minimal "bad" agent: missing fields + stale + unverified + bad behavior
        self.bad_agent = {
            "id": "agent_bad",
            "name": "Bad Agent",
            "url": "",
            "capabilities": [],
            "updated_at": "2023-01-01",
            "domain_verified": False,
            "key_present": False,
            "handshake_fail_ratio": 0.60,
            "rate_limit_violations": 40,
            "complaint_flags": 10,
        }

        # Two agents with identical score inputs except name/id to test tie-break stability
        self.tie_a = {
            "id": "agent_tie_a",
            "name": "Alpha",
            "url": "https://tie.example/a",
            "capabilities": ["info"],
            "contact": "a@tie.example",
            "updated_at": "2025-12-28",
            "domain_verified": True,
            "key_present": True,
            "handshake_fail_ratio": 0.01,
            "rate_limit_violations": 0,
            "complaint_flags": 0,
        }
        self.tie_b = dict(self.tie_a)
        self.tie_b["id"] = "agent_tie_b"
        self.tie_b["name"] = "Beta"
        self.tie_b["url"] = "https://tie.example/b"

    def test_good_agent_scores_high(self):
        ranked = rank_agents([self.good_agent])
        trust = ranked[0].get("trust", {})
        self.assertIn("score", trust)
        self.assertIn("band", trust)
        self.assertIn("reasons", trust)

        # "High" threshold. Adjust if you change weights later.
        self.assertGreaterEqual(trust["score"], 80.0)
        self.assertEqual(trust["band"], "green")
        self.assertTrue(isinstance(trust["reasons"], list))
        self.assertGreaterEqual(len(trust["reasons"]), 1)

    def test_bad_agent_scores_low(self):
        ranked = rank_agents([self.bad_agent])
        trust = ranked[0].get("trust", {})
        self.assertLessEqual(trust["score"], 49.9)
        self.assertEqual(trust["band"], "red")

    def test_ranking_orders_by_score_desc(self):
        ranked = rank_agents([self.bad_agent, self.good_agent])
        self.assertEqual(ranked[0]["id"], "agent_good")
        self.assertEqual(ranked[-1]["id"], "agent_bad")

        # Explicit score ordering check
        top_score = ranked[0]["trust"]["score"]
        bottom_score = ranked[-1]["trust"]["score"]
        self.assertGreaterEqual(top_score, bottom_score)

    def test_deterministic_tie_break(self):
        # With identical scores, we expect stable ordering:
        # score desc, then name asc, then id asc (per your sort key).
        ranked = rank_agents([self.tie_b, self.tie_a])
        self.assertEqual(ranked[0]["id"], "agent_tie_a")
        self.assertEqual(ranked[1]["id"], "agent_tie_b")

        # Run again to ensure repeatability
        ranked2 = rank_agents([self.tie_b, self.tie_a])
        self.assertEqual([a["id"] for a in ranked], [a["id"] for a in ranked2])


if __name__ == "__main__":
    unittest.main()
