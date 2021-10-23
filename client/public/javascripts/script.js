(()=>{
    let ws = new WebSocket("ws:host.docker.internal:8001/ws");
    ws.addEventListener('message',(e)=>{
        let data = JSON.parse(e.data);
        if("message" in data) {
            addMessage(data.message);
        }
        console.log(data);
        switch(data.action) {
            case "authentication_successful":
                clearMessages();
                addMessage("Welcome " + data.data.name);
                authForm.parentNode.removeChild(authForm);
                publisherArea.style.display = '';
                ws.send(JSON.stringify({
                    Action: "get_publishers"
                }));

                subscriptionArea.style.display = '';
                break;
            case "your_publishers":
                if(data.data.length > 0) {
                    console.log(data);
                    for(var i = 0, l = data.data.length; i < l; i++) {
                        setupPublisher(data.data[i]);
                    }                    
                }
                break;
            case "publisher_registered":
                addPublisherName.value = '';
                setupPublisher(data.data);
                break;
            case "messages":
                addMessage("received messages...");
                let confirm = [];
                for(let i in data.data) {
                    let message = data.data[i];
                    addMessage("From subscription: " + message.subscription_id);
                    addMessage("Payload");
                    addMessage(message.payload);
                    addMessage("");
                    confirm.push({
                        subscription_id: message.subscription_id,
                        id: message.id
                    });
                }
                sendWSMessage("confirm_messages",null,{
                    messages: confirm
                });
                break;
            case "messages_confirmed":
                addMessage("confirmed receiving " + data.data.confirmed + " messages");
        }
    });
    let wsReady = false;
    ws.onopen = ()=>wsReady = true;

    let authForm = document.getElementById('auth-form'),
        registerArea = document.getElementById('register'),
        authArea = document.getElementById('authenticate'),
        registerCheck = document.getElementById('auth-register'),
        registerName = document.getElementById('auth-name'),
        authID = document.getElementById('auth-id'),
        messageArea = document.getElementById('message-area'),
        addPublisherForm = document.getElementById('add-publisher-form'),
        addPublisherName = document.getElementById('add-publisher-name'),
        publisherArea = document.getElementById('publisher-area'),
        publishersContainer = document.getElementById('publishers-container'),
        subscriptionArea = document.getElementById('subscription-area'),
        subscribeForm = document.getElementById("subscribe-form"),
        subscribeId = document.getElementById("add-subscription-id");

    subscribeForm.addEventListener('submit',(e)=>{
        e.preventDefault();
        if(subscribeId.value == "") return;
        sendWSMessage("subscribe",null,{
            publisher_id: subscribeId.value
        });
    });

    function sendWSMessage(action,message,data) {
        ws.send(JSON.stringify({
            action: action,
            message: message,
            data: data
        }));
    }

    function createFormField(id,label,type,attributes) {
        let formField = document.createElement('div');
        formField.classList.add('c-form-field');
        let fieldLabel = document.createElement('div');
        fieldLabel.classList.add('c-form-field__label');
        let labelEl = document.createElement('label');
        labelEl.innerHTML = label;
        labelEl.for = id;
        fieldLabel.appendChild(labelEl);
        formField.appendChild(fieldLabel);
        let fieldField = document.createElement('div');
        fieldField.classList.add('c-form-field__field');
        formField.appendChild(fieldField);
        let input;
        if(type == 'textarea') {
            input = document.createElement('textarea');
        } else {
            input = document.createElement('input');
            input.type = type;
        }
        
        input.id = id;

        if(typeof attributes !== "undefined") {
            for(let key in attributes) {
                input.setAttribute(key,attributes[key]);
            }
        }
        
        fieldField.appendChild(input);
        return formField;
    }


    addPublisherForm.addEventListener('submit',(e)=>{
        e.preventDefault();
        if(addPublisherName.value.trim() === '') return;
        sendWSMessage("register_publisher",null,{
            name: addPublisherName.value
        });
    });

    function publishMessage(e) {
        e.preventDefault();
        let target = e.target;
        let textArea = document.getElementById('message_' + target.dataset.id);
        let ttl = document.getElementById('ttl_' + target.dataset.id);
        if(textArea.value.trim == '' || ttl.value == '' || ttl.value <= 0) return;
        sendWSMessage("publish_message",null,{
            "publisher_id": target.dataset.id,
            "ttl": parseInt(ttl.value),
            "payload": textArea.value
        });
    }
    function setupPublisher(publisher) {
        let container = document.createElement('div');
        container.id = 'publisher_' + publisher.id;
        let head = document.createElement('h3');
        head.innerHTML = publisher.name;
        container.appendChild(head);
        let idHead = document.createElement('h4');
        idHead.innerHTML = "ID: " + publisher.id;
        container.appendChild(idHead);
        let form = document.createElement('form');
        form.dataset.id = publisher.id;
        let messageInput = createFormField("message_" + publisher.id, "Message", "textarea",{
            required: true
        });
        form.appendChild(messageInput);
        let ttlInput = createFormField("ttl_" + publisher.id, "Time to Live", "number",{
            required: true,
            min: 10,
            value: 10
        });
        form.appendChild(ttlInput);
        let submit = document.createElement('input');
        submit.type = 'submit';
        submit.value = "Send Message";
        form.appendChild(submit);
        container.appendChild(form);
        publishersContainer.appendChild(container);
        form.addEventListener('submit', publishMessage);
    }


    function addMessage(message) {
        let msg = document.createElement('p');
        msg.classList.add('ws-message');
        msg.innerHTML = message;
        messageArea.appendChild(msg);
    }

    function clearMessages() {
        messageArea.innerHTML = '';
    }



    registerCheck.addEventListener('change',(e)=>{
        if(registerCheck.checked) {
            registerArea.style.display = '';
            authArea.style.display = 'none';
            registerName.setAttribute('required',"true");
            authID.removeAttribute('required');
        } else {
            registerArea.style.display = 'none';
            authArea.style.display = '';
            authID.setAttribute('required',"true");
            registerName.removeAttribute('required');
        }
    });

    authForm.addEventListener('submit',(e)=>{
        e.preventDefault();
        if(!wsReady) {
            addMessage("Please wait");
        }
        let msg;
        if(registerCheck.checked) {
            msg = JSON.stringify({
                "register": true,
                "name": registerName.value
            });
        } else {
            msg = JSON.stringify({
                "register": false,
                "id": authID.value
            });
        }
        ws.send(msg);
    });

    
})();