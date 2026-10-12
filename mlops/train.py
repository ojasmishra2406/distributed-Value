import joblib
from sklearn.datasets import make_classification
from sklearn.linear_model import LogisticRegression
from sklearn.model_selection import train_test_split

def main():
    print("Generating synthetic dataset...")
    X, y = make_classification(n_samples=1000, n_features=20, random_state=42)
    
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2, random_state=42)
    
    print("Training LogisticRegression model...")
    model = LogisticRegression()
    model.fit(X_train, y_train)
    
    accuracy = model.score(X_test, y_test)
    print(f"Model trained successfully. Accuracy on test set: {accuracy:.4f}")
    
    joblib.dump(model, 'model.joblib')
    print("Model saved to model.joblib")

if __name__ == "__main__":
    main()
