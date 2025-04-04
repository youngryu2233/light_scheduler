
import os
from typing import Optional
from fastapi import FastAPI
from pydantic import BaseModel
import torch
from transformers import AutoModelForCausalLM, AutoTokenizer

app = FastAPI()

# 配置从环境变量读取
MODEL_NAME = os.getenv("MODEL_NAME", "root/Model/")
DEVICE = "cuda" if torch.cuda.is_available() else "cpu"

# 全局加载模型和tokenizer
print(f"Loading model {MODEL_NAME}...")
model = AutoModelForCausalLM.from_pretrained(MODEL_NAME, torch_dtype=torch.float16).to(DEVICE)
tokenizer = AutoTokenizer.from_pretrained(MODEL_NAME)
print("Model loaded successfully")

class InferenceRequest(BaseModel):
    prompt: str
    max_length: Optional[int] = 100
    temperature: Optional[float] = 0.7

@app.post("/generate")
async def generate_text(request: InferenceRequest):
    inputs = tokenizer(request.prompt, return_tensors="pt", device_map="auto")
    
    with torch.no_grad():
        outputs = model.generate(
            **inputs,
            max_length=request.max_length,
            temperature=request.temperature,
            do_sample=True
        )
    
    generated_text = tokenizer.decode(outputs[0], skip_special_tokens=True)
    return {"result": generated_text}

@app.get("/health")
async def health_check():
    return {"status": "healthy", "device": DEVICE}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)