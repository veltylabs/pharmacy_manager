import json

with open('/tmp/comments.json') as f:
    comments = json.load(f)

# we only need to translate English ones
eng_comments = [c for c in comments if any(w in c[2] for w in [' the ', ' is ', ' of ', ' to ', ' a ', ' and ', ' or '])]
print(f"English comments: {len(eng_comments)}")
