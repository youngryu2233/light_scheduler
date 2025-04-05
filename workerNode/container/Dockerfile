FROM pytorch/pytorch:2.3.0-cuda12.1-cudnn8-devel
COPY server.py /app/

# 换源
RUN mkdir -p /root/.config/pip
RUN echo "[global]" > /root/.config/pip/pip.conf && \
    echo "index-url = https://pypi.tuna.tsinghua.edu.cn/simple" >> /root/.config/pip/pip.conf

# 安装依赖包
RUN pip install fastapi
RUN pip install transformers
RUN pip install accelerate
RUN pip install uvicorn
CMD ["python", "/app/server.py"]