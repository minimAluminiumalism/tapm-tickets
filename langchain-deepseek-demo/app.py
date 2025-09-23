import os
import asyncio
from langchain_core.prompts import ChatPromptTemplate

from langchain_deepseek import ChatDeepSeek

prompt_template = """You are a helpful assistant.
Use the following context to answer briefly.

Context:
{context}

Question:
{question}
"""


async def main():

    prompt = ChatPromptTemplate.from_template(prompt_template)

    api_key = os.getenv("OPENAI_API_KEY", "YOUR_API_KEY")
    base_url = os.getenv("OPENAI_BASE_URL", "https://api.deepseek.com/beta")
    model_name = os.getenv("MODEL_NAME", "deepseek-reasoner")

    model = ChatDeepSeek(
        api_base=base_url,
        api_key=api_key,
        model=model_name,
        stream_usage=True,
    )

    chain = prompt | model

    inputs = {"context": "some context", "question": "What's OpenTelemetry?"}
    print("Assistant:", end=" ", flush=True)
    async for chunk in chain.astream(inputs):
        piece = getattr(chunk, "content", "")
        if piece:
            print(piece, end="", flush=True)
    print()


if __name__ == "__main__":
    asyncio.run(main())
