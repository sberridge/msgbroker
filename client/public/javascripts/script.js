(()=>{
    let ws = new WebSocket("ws:localhost:8001/ws");
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
                break;
            case "your_publishers":
                if(data.data.length > 0) {
                    console.log(data);
                    for(var i = 0, l = data.data.length; i < l; i++) {
                        setupPublisher(data.data[i]);
                    }                    
                }
                break;
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
        publisherArea = document.getElementById('publisher-area'),
        publishersContainer = document.getElementById('publishers-container');

    function createFormField(id,label,type) {
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
        
        fieldField.appendChild(input);
        return formField;
    }

    function publishMessage(e) {
        e.preventDefault();
        let target = e.target;
        let textArea = document.getElementById('message_' + target.dataset.id);
        if(textArea.value.trim == '') return;

        ws.send(JSON.stringify({
            "action": "publish_message",
            "data": {
                "publisher_id": target.dataset.id,
                "ttl": 100,
                "payload": textArea.value
            }
        }));
    }
    function setupPublisher(publisher) {
        let container = document.createElement('div');
        container.id = 'publisher_' + publisher.id;
        let head = document.createElement('h3');
        head.innerHTML = publisher.name;
        container.appendChild(head);
        let form = document.createElement('form');
        form.dataset.id = publisher.id;
        let input = createFormField("message_" + publisher.id, "Message", "textarea");
        form.appendChild(input);
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
            authID.removeAttributeNS('required');
        } else {
            registerArea.style.display = 'none';
            authArea.style.display = '';
            authID.setAttribute('required',"true");
            registerName.removeAttributeNS('required');
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